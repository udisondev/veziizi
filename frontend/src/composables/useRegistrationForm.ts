import { ref, computed, watch, reactive } from 'vue'
import type { Country, OrganizationData, OwnerData, RegisterRequest } from '@/types/registration'

const validators = {
  email(value: string): string | null {
    if (!value) return 'Обязательное поле'
    const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return regex.test(value) ? null : 'Некорректный email'
  },

  phone(value: string, country: Country): string | null {
    if (!value) return 'Обязательное поле'
    // Убираем все кроме цифр и +
    const cleaned = value.replace(/[^\d+]/g, '')
    // Проверяем минимальную длину по стране
    const minLengths: Record<Country, number> = {
      RU: 12, // +7 + 10 цифр
      KZ: 12, // +7 + 10 цифр
      BY: 13, // +375 + 9 цифр
    }
    const minLength = minLengths[country]
    if (cleaned.length < minLength) {
      return 'Введите полный номер телефона'
    }
    return null
  },

  inn(value: string, country: Country): string | null {
    if (!value) return 'Обязательное поле'
    const rules: Record<Country, { lengths: number[]; name: string }> = {
      RU: { lengths: [10, 12], name: 'ИНН' },
      KZ: { lengths: [12], name: 'БИН/ИИН' },
      BY: { lengths: [9], name: 'УНП' },
    }
    const rule = rules[country]
    if (!/^\d+$/.test(value)) return `${rule.name} должен содержать только цифры`
    if (!rule.lengths.includes(value.length)) {
      return `${rule.name}: ${rule.lengths.join(' или ')} цифр`
    }
    return null
  },

  password(value: string): string | null {
    if (!value) return 'Обязательное поле'
    if (value.length < 8) return 'Минимум 8 символов'
    return null
  },

  confirmPassword(value: string, password: string): string | null {
    if (!value) return 'Обязательное поле'
    return value === password ? null : 'Пароли не совпадают'
  },

  required(value: string): string | null {
    return value?.trim() ? null : 'Обязательное поле'
  },
}

export const innLabels: Record<Country, string> = {
  RU: 'ИНН',
  KZ: 'БИН/ИИН',
  BY: 'УНП',
}

export const innPlaceholders: Record<Country, string> = {
  RU: '10 или 12 цифр',
  KZ: '12 цифр',
  BY: '9 цифр',
}

export const countryNames: Record<Country, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

export function useRegistrationForm() {
  const currentStep = ref(1)
  const totalSteps = 3

  const organization = reactive<OrganizationData>({
    name: '',
    inn: '',
    legal_name: '',
    country: 'RU',
    phone: '',
    email: '',
    address: '',
  })

  const owner = reactive<OwnerData>({
    owner_name: '',
    owner_email: '',
    owner_phone: '',
    owner_password: '',
    confirm_password: '',
  })

  const errors = reactive<Record<string, string | null>>({})

  function validateStep1(): boolean {
    errors.name = validators.required(organization.name)
    errors.legal_name = validators.required(organization.legal_name)
    errors.inn = validators.inn(organization.inn, organization.country)
    errors.phone = validators.phone(organization.phone, organization.country)
    errors.email = validators.email(organization.email)
    errors.address = validators.required(organization.address)

    return !['name', 'legal_name', 'inn', 'phone', 'email', 'address'].some(
      (key) => errors[key] !== null
    )
  }

  function validateStep2(): boolean {
    errors.owner_name = validators.required(owner.owner_name)
    errors.owner_email = validators.email(owner.owner_email)
    errors.owner_phone = validators.phone(owner.owner_phone, organization.country)
    errors.owner_password = validators.password(owner.owner_password)
    errors.confirm_password = validators.confirmPassword(
      owner.confirm_password,
      owner.owner_password
    )

    return !['owner_name', 'owner_email', 'owner_phone', 'owner_password', 'confirm_password'].some(
      (key) => errors[key] !== null
    )
  }

  function nextStep(): boolean {
    if (currentStep.value === 1 && validateStep1()) {
      currentStep.value = 2
      return true
    }
    if (currentStep.value === 2 && validateStep2()) {
      currentStep.value = 3
      return true
    }
    return false
  }

  function prevStep() {
    if (currentStep.value > 1) {
      currentStep.value--
    }
  }

  function goToStep(step: number) {
    if (step >= 1 && step <= currentStep.value) {
      currentStep.value = step
    }
  }

  const requestData = computed<RegisterRequest>(() => ({
    ...organization,
    owner_email: owner.owner_email,
    owner_password: owner.owner_password,
    owner_name: owner.owner_name,
    owner_phone: owner.owner_phone,
  }))

  watch(
    () => organization.country,
    () => {
      if (organization.inn) {
        errors.inn = validators.inn(organization.inn, organization.country)
      }
    }
  )

  function validateField(field: string) {
    switch (field) {
      case 'name':
      case 'legal_name':
      case 'address':
        errors[field] = validators.required(organization[field as keyof OrganizationData] as string)
        break
      case 'email':
        errors.email = validators.email(organization.email)
        break
      case 'phone':
        errors.phone = validators.phone(organization.phone, organization.country)
        break
      case 'inn':
        errors.inn = validators.inn(organization.inn, organization.country)
        break
      case 'owner_name':
        errors.owner_name = validators.required(owner.owner_name)
        break
      case 'owner_email':
        errors.owner_email = validators.email(owner.owner_email)
        break
      case 'owner_phone':
        errors.owner_phone = validators.phone(owner.owner_phone, organization.country)
        break
      case 'owner_password':
        errors.owner_password = validators.password(owner.owner_password)
        if (owner.confirm_password) {
          errors.confirm_password = validators.confirmPassword(
            owner.confirm_password,
            owner.owner_password
          )
        }
        break
      case 'confirm_password':
        errors.confirm_password = validators.confirmPassword(
          owner.confirm_password,
          owner.owner_password
        )
        break
    }
  }

  return {
    currentStep,
    totalSteps,
    organization,
    owner,
    errors,
    requestData,
    nextStep,
    prevStep,
    goToStep,
    validateField,
    validateStep1,
    validateStep2,
  }
}
