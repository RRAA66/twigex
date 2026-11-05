<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <BaseSelect
        v-if="widget === 'status'"
        :model-value="modelValue"
        :label="label"
        :options="statusOptions"
        :placeholder="placeholder"
        @update:model-value="update"
    />
    <BaseSelect
        v-else-if="widget === 'checkbox'"
        :model-value="modelValue"
        :label="label"
        :options="checkboxOptions"
        :placeholder="placeholder"
        @update:model-value="update"
    />
    <div v-else-if="widget !== 'none'">
        <label :for="id" class="block text-sm font-medium leading-6 text-gray-900">
            {{ label }}
        </label>
        <div class="mt-2">
            <FormLinkedPicker
                v-if="widget === 'linked'"
                :id="id"
                :model-value="picked"
                :loader="loadRows"
                :placeholder="placeholder"
                @update:model-value="pick"
            />
            <FormPersonPicker
                v-else-if="widget === 'assignee'"
                :id="id"
                :model-value="modelValue"
                :workspace-id="workspaceId"
                :placeholder="placeholder"
                @update:model-value="update"
            />
            <DatePickerField
                v-else-if="widget === 'date'"
                :id="id"
                :model-value="modelValue"
                :placeholder="t('projects.form_view.placeholder.pick_date')"
                @update:model-value="update"
            />
            <input
                v-else
                :id="id"
                :value="modelValue"
                type="text"
                maxlength="255"
                class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
                :placeholder="placeholder"
                @input="update($event.target.value)"
            />
        </div>
    </div>
</template>

<script setup>
import { computed, ref, useId, watch } from "vue";
import { t } from "@/i18n/index.js";
import workspaceService from "@/services/workspaceService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import { conditionValueWidget, linkedTableId } from "@/utils/projects/forms";
import BaseSelect from "@/components/BaseSelect.vue";
import DatePickerField from "@/components/DatePicker/DatePickerField.vue";
import { formRowsLoader } from "@/composables/projects/formRowsLoader";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import FormPersonPicker from "@/components/Projects/Form/FormPersonPicker.vue";

const props = defineProps({
    field: { type: Object, required: true },
    operator: { type: String, default: "equals" },
    modelValue: { type: String, default: "" },
    label: { type: String, default: "" },
    workspaceId: { type: String, required: true },
    tableId: { type: String, required: true },
});

const emit = defineEmits(["update:modelValue"]);

const id = useId();
const alertStore = useAlertStore();

const statusOptions = ref([]);
const picked = ref(null);

const widget = computed(() => conditionValueWidget(props.field, props.operator));

const placeholder = computed(() => t.value("projects.form_view.field.condition_value_placeholder"));

const checkboxOptions = computed(() => [
    { value: "true", label: t.value("projects.form_view.field.condition_yes") },
    { value: "false", label: t.value("projects.form_view.field.condition_no") },
]);

function update(value) {
    emit("update:modelValue", value);
}

function pick(item) {
    picked.value = item;
    update(item ? item.id : "");
}

const loadRows = computed(() => formRowsLoader(props.workspaceId, linkedTableId(props.field)));

async function loadStatusOptions() {
    const { data } = await workspaceService.getTableStatusTypes(props.workspaceId, props.tableId);

    statusOptions.value = (data.options || []).map((o) => ({ value: o.id, label: o.name }));
}

// A saved value is a row id; the picker shows its name.
async function rowName(value) {
    const { data } = await workspaceService.getLinkedRecordsLite(
        props.workspaceId,
        linkedTableId(props.field),
        { ids: value },
    );

    return data?.[0]?.name;
}

async function showError(load) {
    try {
        await load();
    } catch (error) {
        alertStore.showError(extractErrorMessage(error));
    }
}

watch(
    widget,
    (kind) => {
        if (kind === "status") showError(loadStatusOptions);
    },
    { immediate: true },
);

// Follows the value, so one cleared elsewhere, as by an operator that takes
// none, is not still shown as picked.
watch(
    [widget, () => props.modelValue],
    ([kind, value]) => {
        if (kind !== "linked") return;

        if (!value) {
            picked.value = null;

            return;
        }

        if (picked.value?.id === value) return;

        picked.value = { id: value, name: value };
        showError(async () => {
            const name = await rowName(value);

            if (name && picked.value?.id === value) picked.value = { id: value, name };
        });
    },
    { immediate: true },
);
</script>
