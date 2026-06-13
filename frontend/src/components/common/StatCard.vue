<template>
  <div
    ref="rootRef"
    class="stat-card"
    @mouseenter="handleHover(true)"
    @mouseleave="handleHover(false)"
  >
    <div :class="['stat-card-motion-item', 'stat-icon', iconClass]">
      <component v-if="icon" :is="icon" class="h-6 w-6" aria-hidden="true" />
    </div>
    <div class="stat-card-motion-item min-w-0 flex-1">
      <p class="stat-label truncate">{{ title }}</p>
      <div class="mt-1 flex items-baseline gap-2">
        <p class="stat-value" :title="String(formattedValue)">{{ formattedValue }}</p>
        <span v-if="change !== undefined" :class="['stat-trend', trendClass]">
          <Icon
            v-if="changeType !== 'neutral'"
            name="arrowUp"
            size="xs"
            :class="changeType === 'down' && 'rotate-180'"
          />
          {{ formattedChange }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { Component } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { animateHoverLift, animateMountedSurface, clearMotion } from '@/composables/useGsapMotion'

const rootRef = ref<HTMLElement | null>(null)

type ChangeType = 'up' | 'down' | 'neutral'
type IconVariant = 'primary' | 'success' | 'warning' | 'danger'

interface Props {
  title: string
  value: number | string
  icon?: Component
  iconVariant?: IconVariant
  change?: number
  changeType?: ChangeType
  formatValue?: (value: number | string) => string
}

const props = withDefaults(defineProps<Props>(), {
  changeType: 'neutral',
  iconVariant: 'primary'
})

const formattedValue = computed(() => {
  if (props.formatValue) {
    return props.formatValue(props.value)
  }
  if (typeof props.value === 'number') {
    return props.value.toLocaleString()
  }
  return props.value
})

const formattedChange = computed(() => {
  if (props.change === undefined) return ''
  const absChange = Math.abs(props.change)
  return `${absChange}%`
})

const iconClass = computed(() => {
  const classes: Record<IconVariant, string> = {
    primary: 'stat-icon-primary',
    success: 'stat-icon-success',
    warning: 'stat-icon-warning',
    danger: 'stat-icon-danger'
  }
  return classes[props.iconVariant]
})

const trendClass = computed(() => {
  const classes: Record<ChangeType, string> = {
    up: 'stat-trend-up',
    down: 'stat-trend-down',
    neutral: 'text-gray-500 dark:text-dark-400'
  }
  return classes[props.changeType]
})

function handleHover(lifted: boolean) {
  if (rootRef.value) {
    animateHoverLift(rootRef.value, lifted)
  }
}

onMounted(() => {
  if (rootRef.value) {
    animateMountedSurface(rootRef.value, '.stat-card-motion-item')
  }
})

onUnmounted(() => {
  if (rootRef.value) {
    clearMotion(rootRef.value)
  }
})
</script>
