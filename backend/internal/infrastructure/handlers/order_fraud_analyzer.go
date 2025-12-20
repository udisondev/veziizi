package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// Order fraud signal types
const (
	SignalCancelAfterAccept = "cancel_after_accept"
	SignalGhostDelivery     = "ghost_deliveries"
	SignalCircularOrders    = "circular_orders"
)

// Order fraud thresholds
var OrderFraudThresholds = struct {
	CancelRateThreshold      float64 // >30% cancellation rate
	MinOrdersForCancelRate   int     // minimum orders to check cancel rate
	CancelWithinHours        int     // cancel within X hours is suspicious
	GhostDeliveryMinKm       int     // minimum distance to check for ghost
	GhostDeliverySpeedKmH    int     // max realistic speed km/h
	CircularOrdersDays       int     // check circular orders within X days
	CircularOrdersMinCount   int     // minimum orders in each direction
}{
	CancelRateThreshold:      0.3,
	MinOrdersForCancelRate:   5,
	CancelWithinHours:        24,
	GhostDeliveryMinKm:       100,
	GhostDeliverySpeedKmH:    80,
	CircularOrdersDays:       30,
	CircularOrdersMinCount:   2,
}

// OrderFraudAnalyzerHandler listens for Order events
// and performs fraud detection
type OrderFraudAnalyzerHandler struct {
	orderFraud *projections.OrderFraudProjection
	orders     *projections.OrdersProjection
}

func NewOrderFraudAnalyzerHandler(
	orderFraud *projections.OrderFraudProjection,
	orders *projections.OrdersProjection,
) *OrderFraudAnalyzerHandler {
	return &OrderFraudAnalyzerHandler{
		orderFraud: orderFraud,
		orders:     orders,
	}
}

func (h *OrderFraudAnalyzerHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event: %w", err)
	}

	ctx := msg.Context()

	switch e := evt.(type) {
	case orderEvents.OrderCreated:
		return h.onOrderCreated(ctx, e)
	case orderEvents.OrderCancelled:
		return h.onOrderCancelled(ctx, e, envelope.AggregateID)
	case orderEvents.OrderCompleted:
		return h.onOrderCompleted(ctx, e, envelope.AggregateID)
	default:
		// Ignore other order events
		return nil
	}
}

func (h *OrderFraudAnalyzerHandler) onOrderCreated(ctx context.Context, e orderEvents.OrderCreated) error {
	slog.Info("order fraud: analyzing created order",
		slog.String("order_id", e.AggregateID().String()),
		slog.String("customer_org_id", e.CustomerOrgID.String()),
		slog.String("carrier_org_id", e.CarrierOrgID.String()),
	)

	// Increment order counts
	if err := h.orderFraud.IncrementOrderCreated(ctx, e.CustomerOrgID, e.CarrierOrgID); err != nil {
		slog.Error("failed to increment order counts", slog.String("error", err.Error()))
		// Don't fail the handler, just log
	}

	// Check for circular orders
	if err := h.checkCircularOrders(ctx, e.AggregateID(), e.CustomerOrgID, e.CarrierOrgID); err != nil {
		slog.Warn("failed to check circular orders", slog.String("error", err.Error()))
	}

	return nil
}

func (h *OrderFraudAnalyzerHandler) onOrderCancelled(ctx context.Context, e orderEvents.OrderCancelled, orderID uuid.UUID) error {
	slog.Info("order fraud: analyzing cancelled order",
		slog.String("order_id", orderID.String()),
		slog.String("cancelled_by", e.CancelledByOrgID.String()),
	)

	// Get order details
	order, err := h.orders.GetByID(ctx, orderID)
	if err != nil {
		slog.Error("failed to get order", slog.String("error", err.Error()))
		return nil // Don't fail handler
	}

	// Determine if cancelled as customer or carrier
	asCustomer := e.CancelledByOrgID == order.CustomerOrgID

	// Increment cancellation count
	if err := h.orderFraud.IncrementOrderCancelled(ctx, e.CancelledByOrgID, asCustomer); err != nil {
		slog.Error("failed to increment cancelled count", slog.String("error", err.Error()))
	}

	// Check cancel rate
	if err := h.checkCancelAfterAccept(ctx, orderID, e.CancelledByOrgID, order.CreatedAt, e.OccurredAt()); err != nil {
		slog.Warn("failed to check cancel after accept", slog.String("error", err.Error()))
	}

	return nil
}

func (h *OrderFraudAnalyzerHandler) onOrderCompleted(ctx context.Context, e orderEvents.OrderCompleted, orderID uuid.UUID) error {
	slog.Info("order fraud: analyzing completed order",
		slog.String("order_id", orderID.String()),
	)

	// Get order details
	order, err := h.orders.GetByID(ctx, orderID)
	if err != nil {
		slog.Error("failed to get order", slog.String("error", err.Error()))
		return nil
	}

	// Calculate completion time
	completionHours := e.OccurredAt().Sub(order.CreatedAt).Hours()

	// Update completion stats
	if err := h.orderFraud.IncrementOrderCompleted(ctx, order.CustomerOrgID, order.CarrierOrgID, completionHours); err != nil {
		slog.Error("failed to increment completed count", slog.String("error", err.Error()))
	}

	// Check for ghost delivery
	if err := h.checkGhostDelivery(ctx, orderID, order.CustomerOrgID, order.CarrierOrgID, completionHours); err != nil {
		slog.Warn("failed to check ghost delivery", slog.String("error", err.Error()))
	}

	return nil
}

// checkCancelAfterAccept checks if org has high cancellation rate
func (h *OrderFraudAnalyzerHandler) checkCancelAfterAccept(ctx context.Context, orderID, orgID uuid.UUID, orderCreatedAt, cancelledAt time.Time) error {
	// Check if cancelled quickly after creation
	hoursToCancel := cancelledAt.Sub(orderCreatedAt).Hours()
	quickCancel := hoursToCancel < float64(OrderFraudThresholds.CancelWithinHours)

	// Get cancel rates
	customerRate, carrierRate, err := h.orderFraud.GetCancelRate(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get cancel rate: %w", err)
	}

	behavior, err := h.orderFraud.GetOrgOrderBehavior(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get org behavior: %w", err)
	}

	// Check if enough orders to make a determination
	totalOrders := behavior.TotalOrdersAsCustomer + behavior.TotalOrdersAsCarrier
	if totalOrders < OrderFraudThresholds.MinOrdersForCancelRate {
		return nil
	}

	// Check if cancel rate exceeds threshold
	maxRate := max(customerRate, carrierRate)
	if maxRate >= OrderFraudThresholds.CancelRateThreshold || quickCancel {
		signal := &projections.OrderFraudSignal{
			OrderID:     orderID,
			OrgID:       orgID,
			SignalType:  SignalCancelAfterAccept,
			Severity:    "high",
			Description: fmt.Sprintf("High cancellation rate: %.0f%% (customer: %.0f%%, carrier: %.0f%%), cancelled in %.1f hours", maxRate*100, customerRate*100, carrierRate*100, hoursToCancel),
			ScoreImpact: 0.35,
			Evidence:    fmt.Sprintf(`{"customer_rate": %.2f, "carrier_rate": %.2f, "hours_to_cancel": %.1f, "quick_cancel": %t}`, customerRate, carrierRate, hoursToCancel, quickCancel),
		}

		if err := h.orderFraud.InsertOrderFraudSignal(ctx, signal); err != nil {
			return fmt.Errorf("insert fraud signal: %w", err)
		}

		// Mark org as suspicious if rate is very high
		if maxRate >= 0.5 {
			behavior.IsSuspicious = true
			reason := fmt.Sprintf("Cancellation rate %.0f%% exceeds threshold", maxRate*100)
			behavior.SuspiciousReason = &reason
			if err := h.orderFraud.UpsertOrgOrderBehavior(ctx, behavior); err != nil {
				slog.Error("failed to mark org suspicious", slog.String("error", err.Error()))
			}
		}

		slog.Info("order fraud signal detected: cancel_after_accept",
			slog.String("order_id", orderID.String()),
			slog.String("org_id", orgID.String()),
			slog.Float64("cancel_rate", maxRate),
		)
	}

	return nil
}

// checkGhostDelivery checks if order was completed unrealistically fast
func (h *OrderFraudAnalyzerHandler) checkGhostDelivery(ctx context.Context, orderID, customerOrgID, carrierOrgID uuid.UUID, completionHours float64) error {
	// Get average completion time for this org
	behavior, err := h.orderFraud.GetOrgOrderBehavior(ctx, carrierOrgID)
	if err != nil {
		return fmt.Errorf("get org behavior: %w", err)
	}

	// If this is much faster than average, flag it
	// Also check minimum completion hours
	isGhost := false
	var reason string

	if behavior.MinCompletionHours != nil && completionHours < *behavior.MinCompletionHours*0.5 {
		isGhost = true
		reason = fmt.Sprintf("Completed in %.1f hours, min for this carrier is %.1f hours", completionHours, *behavior.MinCompletionHours)
	}

	if behavior.AvgCompletionHours != nil && completionHours < *behavior.AvgCompletionHours*0.25 {
		isGhost = true
		reason = fmt.Sprintf("Completed in %.1f hours, avg for this carrier is %.1f hours", completionHours, *behavior.AvgCompletionHours)
	}

	// Absolute threshold: less than 1 hour is always suspicious for logistics
	if completionHours < 1 {
		isGhost = true
		reason = fmt.Sprintf("Completed in %.1f hours - impossibly fast for logistics", completionHours)
	}

	if isGhost {
		signal := &projections.OrderFraudSignal{
			OrderID:     orderID,
			OrgID:       carrierOrgID,
			SignalType:  SignalGhostDelivery,
			Severity:    "high",
			Description: reason,
			ScoreImpact: 0.4,
			Evidence:    fmt.Sprintf(`{"completion_hours": %.2f, "avg_hours": %v, "min_hours": %v}`, completionHours, behavior.AvgCompletionHours, behavior.MinCompletionHours),
		}

		if err := h.orderFraud.InsertOrderFraudSignal(ctx, signal); err != nil {
			return fmt.Errorf("insert fraud signal: %w", err)
		}

		slog.Info("order fraud signal detected: ghost_delivery",
			slog.String("order_id", orderID.String()),
			slog.Float64("completion_hours", completionHours),
		)
	}

	return nil
}

// checkCircularOrders checks for suspicious circular order patterns
func (h *OrderFraudAnalyzerHandler) checkCircularOrders(ctx context.Context, orderID, customerOrgID, carrierOrgID uuid.UUID) error {
	data, err := h.orderFraud.GetCircularOrderData(ctx, customerOrgID, carrierOrgID, OrderFraudThresholds.CircularOrdersDays)
	if err != nil {
		return fmt.Errorf("get circular order data: %w", err)
	}

	// Check if there are orders in both directions
	if data.OrdersAToB >= OrderFraudThresholds.CircularOrdersMinCount &&
		data.OrdersBToA >= OrderFraudThresholds.CircularOrdersMinCount {

		signal := &projections.OrderFraudSignal{
			OrderID:     orderID,
			OrgID:       customerOrgID,
			SignalType:  SignalCircularOrders,
			Severity:    "high",
			Description: fmt.Sprintf("Circular orders detected: %d orders A→B, %d orders B→A in last %d days", data.OrdersAToB, data.OrdersBToA, OrderFraudThresholds.CircularOrdersDays),
			ScoreImpact: 0.5,
			Evidence:    fmt.Sprintf(`{"orders_a_to_b": %d, "orders_b_to_a": %d, "days": %d}`, data.OrdersAToB, data.OrdersBToA, OrderFraudThresholds.CircularOrdersDays),
		}

		if err := h.orderFraud.InsertOrderFraudSignal(ctx, signal); err != nil {
			return fmt.Errorf("insert fraud signal: %w", err)
		}

		// Also create signal for carrier
		signal.OrgID = carrierOrgID
		if err := h.orderFraud.InsertOrderFraudSignal(ctx, signal); err != nil {
			return fmt.Errorf("insert fraud signal for carrier: %w", err)
		}

		// Record chain
		chain := &projections.OrderChain{
			ChainOrgs:    []uuid.UUID{customerOrgID, carrierOrgID},
			ChainLength:  2,
			OrderIDs:     []uuid.UUID{orderID},
			IsSuspicious: true,
		}
		if data.LastOrderAToB != nil {
			chain.FirstOrderAt = *data.LastOrderAToB
		}
		if data.LastOrderBToA != nil {
			chain.LastOrderAt = *data.LastOrderBToA
		}
		if chain.FirstOrderAt.IsZero() {
			chain.FirstOrderAt = time.Now()
		}
		if chain.LastOrderAt.IsZero() {
			chain.LastOrderAt = time.Now()
		}

		if err := h.orderFraud.InsertOrderChain(ctx, chain); err != nil {
			slog.Warn("failed to insert order chain", slog.String("error", err.Error()))
		}

		slog.Info("order fraud signal detected: circular_orders",
			slog.String("order_id", orderID.String()),
			slog.String("customer_org", customerOrgID.String()),
			slog.String("carrier_org", carrierOrgID.String()),
		)
	}

	return nil
}
