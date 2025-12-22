<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import 'leaflet/dist/leaflet.css'
import L from 'leaflet'
import type { Coordinates, RoutePoint } from '@/types/freightRequest'

// Fix for default markers in Leaflet with bundlers
import iconUrl from 'leaflet/dist/images/marker-icon.png'
import iconRetinaUrl from 'leaflet/dist/images/marker-icon-2x.png'
import shadowUrl from 'leaflet/dist/images/marker-shadow.png'

delete (L.Icon.Default.prototype as any)._getIconUrl
L.Icon.Default.mergeOptions({
  iconUrl,
  iconRetinaUrl,
  shadowUrl,
})

interface Props {
  points: RoutePoint[]
  height?: string
  interactive?: boolean
  navigable?: boolean
}

interface Emits {
  (e: 'click', coordinates: Coordinates): void
}

const props = withDefaults(defineProps<Props>(), {
  height: '300px',
  interactive: false,
  navigable: true,
})

const emit = defineEmits<Emits>()

const mapContainer = ref<HTMLDivElement | null>(null)
let map: L.Map | null = null
let markersLayer: L.LayerGroup | null = null
let polyline: L.Polyline | null = null

// Inline стили для маркеров (CSS классы могут не работать с Leaflet divIcon)
const markerPinStyle = `
  width: 20px;
  height: 20px;
  border-radius: 50% 50% 50% 0;
  transform: rotate(-45deg);
  margin: 5px;
`

// Маркер только погрузка (синий)
const loadingIcon = L.divIcon({
  className: 'custom-marker',
  html: `<div style="${markerPinStyle} background-color: #3b82f6;"></div>`,
  iconSize: [30, 30],
  iconAnchor: [15, 30],
})

// Маркер только разгрузка (зелёный)
const unloadingIcon = L.divIcon({
  className: 'custom-marker',
  html: `<div style="${markerPinStyle} background-color: #22c55e;"></div>`,
  iconSize: [30, 30],
  iconAnchor: [15, 30],
})

// Маркер и погрузка и разгрузка (двухцветный)
const dualIcon = L.divIcon({
  className: 'custom-marker',
  html: `<div style="${markerPinStyle} overflow: hidden; position: relative;">
    <div style="position: absolute; left: 0; top: 0; width: 50%; height: 100%; background-color: #3b82f6;"></div>
    <div style="position: absolute; right: 0; top: 0; width: 50%; height: 100%; background-color: #22c55e;"></div>
  </div>`,
  iconSize: [30, 30],
  iconAnchor: [15, 30],
})

// Маркер без типа (серый)
const neutralIcon = L.divIcon({
  className: 'custom-marker',
  html: `<div style="${markerPinStyle} background-color: #9ca3af;"></div>`,
  iconSize: [30, 30],
  iconAnchor: [15, 30],
})

function getMarkerIcon(point: RoutePoint): L.DivIcon {
  if (point.is_loading && point.is_unloading) {
    return dualIcon
  }
  if (point.is_loading) {
    return loadingIcon
  }
  if (point.is_unloading) {
    return unloadingIcon
  }
  return neutralIcon
}

function getPointTypeLabel(point: RoutePoint): string {
  if (point.is_loading && point.is_unloading) {
    return 'Погрузка/Разгрузка'
  }
  if (point.is_loading) {
    return 'Погрузка'
  }
  if (point.is_unloading) {
    return 'Разгрузка'
  }
  return 'Точка'
}

const validPoints = computed(() =>
  props.points.filter((p) => p.coordinates)
)

function updateMarkers() {
  if (!map || !markersLayer) return

  markersLayer.clearLayers()

  if (polyline) {
    map.removeLayer(polyline)
    polyline = null
  }

  const coords: L.LatLng[] = []

  validPoints.value.forEach((point, index) => {
    if (!point.coordinates) return

    const latLng = L.latLng(point.coordinates.latitude, point.coordinates.longitude)
    coords.push(latLng)

    const icon = getMarkerIcon(point)
    const marker = L.marker(latLng, { icon })

    marker.bindPopup(`
      <div class="text-sm">
        <div class="font-medium">${getPointTypeLabel(point)} #${index + 1}</div>
        <div class="text-gray-600">${point.address || 'Адрес не указан'}</div>
      </div>
    `)

    markersLayer?.addLayer(marker)
  })

  // Draw route line
  if (coords.length > 1) {
    polyline = L.polyline(coords, {
      color: '#3b82f6',
      weight: 3,
      opacity: 0.7,
      dashArray: '10, 10',
    })
    polyline.addTo(map)
  }

  // Fit bounds
  if (coords.length > 0) {
    const bounds = L.latLngBounds(coords)
    map.fitBounds(bounds, { padding: [50, 50], maxZoom: 12 })
  }
}

function initMap() {
  if (!mapContainer.value || map) return

  // Default center - Moscow
  const defaultCenter: L.LatLngExpression = [55.7558, 37.6173]

  map = L.map(mapContainer.value, {
    center: defaultCenter,
    zoom: 5,
    scrollWheelZoom: props.navigable,
    dragging: props.navigable,
    touchZoom: props.navigable,
    doubleClickZoom: props.navigable,
    boxZoom: props.navigable,
  })

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution:
      '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
  }).addTo(map)

  markersLayer = L.layerGroup().addTo(map)

  if (props.interactive) {
    map.on('click', (e: L.LeafletMouseEvent) => {
      emit('click', {
        latitude: e.latlng.lat,
        longitude: e.latlng.lng,
      })
    })
  }

  updateMarkers()
}

onMounted(() => {
  initMap()
})

watch(
  () => props.points,
  () => {
    updateMarkers()
  },
  { deep: true }
)
</script>

<template>
  <div class="relative isolate">
    <div
      ref="mapContainer"
      :style="{ height }"
      class="w-full rounded-lg border border-gray-200 overflow-hidden"
    />
  </div>
</template>

<style>
.custom-marker .marker-pin {
  width: 20px;
  height: 20px;
  border-radius: 50% 50% 50% 0;
  transform: rotate(-45deg);
  margin: 5px;
}

.custom-marker .marker-pin.bg-blue {
  background-color: #3b82f6;
}

.custom-marker .marker-pin.bg-green {
  background-color: #22c55e;
}

.custom-marker .marker-pin.bg-gray {
  background-color: #9ca3af;
}

/* Двухцветный маркер */
.custom-marker .marker-pin-dual {
  width: 20px;
  height: 20px;
  border-radius: 50% 50% 50% 0;
  transform: rotate(-45deg);
  margin: 5px;
  overflow: hidden;
  position: relative;
}

.custom-marker .marker-pin-dual .half-left {
  position: absolute;
  left: 0;
  top: 0;
  width: 50%;
  height: 100%;
  background-color: #3b82f6;
}

.custom-marker .marker-pin-dual .half-right {
  position: absolute;
  right: 0;
  top: 0;
  width: 50%;
  height: 100%;
  background-color: #22c55e;
}

.leaflet-popup-content-wrapper {
  border-radius: 8px;
}

.leaflet-popup-content {
  margin: 8px 12px;
}
</style>
