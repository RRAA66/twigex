<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="border-b border-gray-100 last:border-b-0">
        <div class="flex items-center justify-between gap-x-3 px-4 py-2 hover:bg-gray-50">
            <div class="flex min-w-0 flex-1 items-center gap-x-1.5">
                <Bars2Icon
                    class="form-field-drag-handle h-4 w-4 shrink-0 cursor-grab text-gray-300"
                    aria-hidden="true"
                />
                <button
                    type="button"
                    class="flex min-w-0 flex-1 select-none items-center gap-x-1.5 rounded text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-600"
                    :aria-expanded="expanded"
                    :aria-controls="`${id}-settings`"
                    @click="expanded = !expanded"
                >
                    <ChevronRightIcon
                        :class="[
                            'h-3 w-3 shrink-0 text-gray-400 transition-transform',
                            expanded ? 'rotate-90' : '',
                        ]"
                        aria-hidden="true"
                    />
                    <span
                        :class="[
                            'truncate text-sm',
                            field.visible ? 'text-gray-700' : 'text-gray-300 line-through',
                        ]"
                    >
                        {{ field.label || fieldName }}
                    </span>
                    <span v-if="field.required" class="shrink-0 text-xs text-red-400">*</span>
                </button>
            </div>
            <ToggleSwitch
                v-if="!isNameField"
                v-model="field.visible"
                :sr-label="t('projects.form_view.field.visible')"
            />
        </div>

        <div
            v-if="expanded"
            :id="`${id}-settings`"
            class="grid grid-cols-2 gap-3 border-t border-gray-200 bg-gray-50 p-4"
        >
            <div class="col-span-2">
                <label :for="`${id}-label`" :class="labelClass">
                    {{ t("projects.form_view.field.label") }}
                </label>
                <input
                    :id="`${id}-label`"
                    v-model="field.label"
                    type="text"
                    :class="inputClass"
                    :placeholder="fieldName"
                />
            </div>
            <div class="col-span-2">
                <label :for="`${id}-placeholder`" :class="labelClass">
                    {{ t("projects.form_view.field.placeholder_label") }}
                </label>
                <input
                    :id="`${id}-placeholder`"
                    v-model="field.placeholder"
                    type="text"
                    :class="inputClass"
                    :placeholder="t('projects.form_view.placeholder.your_answer')"
                />
            </div>
            <BaseSelect
                v-model="field.column_position"
                :label="t('projects.form_view.field.column_width')"
                :options="columnOptions"
            />
            <div v-if="!isNameField" class="flex items-end pb-1.5">
                <BaseCheckbox v-model="field.required" :label="t('projects.form_view.required')" />
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed, ref, useId } from "vue";
import { Bars2Icon, ChevronRightIcon } from "@heroicons/vue/24/outline";
import { t } from "@/i18n/index.js";
import BaseCheckbox from "@/components/BaseCheckbox.vue";
import BaseSelect from "@/components/BaseSelect.vue";
import ToggleSwitch from "@/components/ToggleSwitch.vue";

const props = defineProps({
    field: {
        type: Object,
        required: true,
    },
    fieldName: {
        type: String,
        default: "",
    },
});

const id = useId();

const labelClass = "block text-sm font-medium leading-6 text-gray-900";
const inputClass =
    "mt-2 block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6";

const expanded = ref(false);

const isNameField = computed(() => props.field.header_name === "name");

const columnOptions = computed(() => [
    { value: "full", label: t.value("projects.form_view.field.col_full") },
    { value: "left", label: t.value("projects.form_view.field.col_left") },
    { value: "right", label: t.value("projects.form_view.field.col_right") },
]);
</script>
