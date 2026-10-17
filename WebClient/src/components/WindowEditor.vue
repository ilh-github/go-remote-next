<template>
  <div class="window-editor">
    <div v-if="modelValue.length === 0" class="window-empty">{{ emptyLabel }}</div>
    <div v-for="(window, index) in modelValue" :key="index" class="window-row">
      <div class="day-picker" role="group" :aria-label="$t('notifications.windows.days')">
        <label
          v-for="day in days"
          :key="day.value"
          :class="['day-toggle', { active: window.days.includes(day.value) }]"
          :title="day.long"
        >
          <input
            type="checkbox"
            :checked="window.days.includes(day.value)"
            @change="toggleDay(index, day.value, $event.target.checked)"
          />
          <span>{{ day.short }}</span>
        </label>
      </div>
      <div class="time-range">
        <input
          class="form-input time-input"
          type="time"
          :value="window.start"
          @input="updateWindow(index, 'start', $event.target.value)"
        />
        <span class="range-separator">-</span>
        <input
          class="form-input time-input"
          type="time"
          :value="window.end"
          @input="updateWindow(index, 'end', $event.target.value)"
        />
      </div>
      <button
        class="remove-window"
        type="button"
        :title="$t('notifications.windows.remove')"
        @click="removeWindow(index)"
      >✕</button>
    </div>
    <button class="btn btn-secondary btn-sm add-window" type="button" @click="addWindow">
      {{ addLabel || $t('notifications.windows.add') }}
    </button>
    <p v-if="hint" class="hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  addLabel: { type: String, default: '' },
  emptyLabel: { type: String, default: '' },
  hint: { type: String, default: '' },
  defaultStart: { type: String, default: '22:00' },
  defaultEnd: { type: String, default: '07:00' },
})

const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()

const days = computed(() => [
  { value: 1, short: t('notifications.days.mon'), long: t('notifications.days.monday') },
  { value: 2, short: t('notifications.days.tue'), long: t('notifications.days.tuesday') },
  { value: 3, short: t('notifications.days.wed'), long: t('notifications.days.wednesday') },
  { value: 4, short: t('notifications.days.thu'), long: t('notifications.days.thursday') },
  { value: 5, short: t('notifications.days.fri'), long: t('notifications.days.friday') },
  { value: 6, short: t('notifications.days.sat'), long: t('notifications.days.saturday') },
  { value: 0, short: t('notifications.days.sun'), long: t('notifications.days.sunday') },
])

function emitRows(rows) {
  emit('update:modelValue', rows)
}

function toggleDay(index, day, checked) {
  emitRows(props.modelValue.map((window, rowIndex) => {
    if (rowIndex !== index) return window
    const next = new Set(window.days)
    if (checked) next.add(day)
    else next.delete(day)
    return { ...window, days: [...next].sort((a, b) => a - b) }
  }))
}

function updateWindow(index, field, value) {
  emitRows(props.modelValue.map((window, rowIndex) => (
    rowIndex === index ? { ...window, [field]: value } : window
  )))
}

function addWindow() {
  emitRows([
    ...props.modelValue,
    {
      days: [1, 2, 3, 4, 5],
      start: props.defaultStart,
      end: props.defaultEnd,
    },
  ])
}

function removeWindow(index) {
  emitRows(props.modelValue.filter((_, rowIndex) => rowIndex !== index))
}
</script>

<style scoped>
.window-editor {
  display: grid;
  gap: 10px;
}

.window-empty {
  padding: 12px;
  border: 1px dashed rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 13px;
}

.window-row {
  display: grid;
  grid-template-columns: minmax(230px, 1fr) auto 32px;
  gap: 10px;
  align-items: center;
  padding: 10px;
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 8px;
}

.day-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.day-toggle {
  position: relative;
  min-width: 32px;
  padding: 5px 7px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  text-align: center;
  cursor: pointer;
  user-select: none;
}

.day-toggle input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.day-toggle.active {
  border-color: rgba(76, 175, 80, 0.55);
  background: rgba(76, 175, 80, 0.13);
  color: #b8edbb;
}

.time-range {
  display: flex;
  align-items: center;
  gap: 6px;
}

.time-input {
  width: 116px;
  padding: 7px 9px;
}

.range-separator {
  color: var(--text-secondary);
}

.remove-window {
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 6px;
  background: rgba(244, 67, 54, 0.1);
  color: #ff9b94;
  cursor: pointer;
}

.remove-window:hover {
  background: rgba(244, 67, 54, 0.2);
}

.add-window {
  justify-self: start;
}

@media (max-width: 760px) {
  .window-row {
    grid-template-columns: 1fr;
  }

  .remove-window {
    justify-self: end;
  }
}
</style>
