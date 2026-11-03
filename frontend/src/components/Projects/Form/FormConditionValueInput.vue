<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <BaseSelect
        v-if="widget === 'status'"
        :model-value="modelValue"
        :options="statusOptions"
        :placeholder="placeholder"
        @update:model-value="update"
    />
    <FormLinkedPicker
        v-else-if="widget === 'linked'"
        :model-value="picked"
        :workspace-id="workspaceId"
        :table-id="linkedTableId(field)"
        :placeholder="placeholder"
        @update:model-value="pick"
    />
    <FormMemberPicker
        v-else-if="widget === 'assignee'"
        :model-value="modelValue"
        :members="members"
        :placeholder="placeholder"
        @update:model-value="update"
    />
    <DatePickerField
        v-else-if="widget === 'date'"
        :model-value="modelValue"
        :placeholder="t('projects.form_view.placeholder.pick_date')"
        @update:model-value="update"
    />
    <BaseSelect
        v-else-if="widget === 'checkbox'"
        :model-value="modelValue"
        :options="checkboxOptions"
        :placeholder="placeholder"
        @update:model-value="update"
    />
    <input
        v-else-if="widget === 'text'"
        :value="modelValue"
        type="text"
        maxlength="255"
        class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
        :placeholder="placeholder"
        @input="update($event.target.value)"
    />
</template>

<script setup>
import { computed, ref, watch } from "vue";
import { t } from "@/i18n/index.js";
import workspaceService from "@/services/workspaceService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import { conditionValueWidget } from "@/utils/projects/conditionValueWidget";
import { linkedTableId } from "@/utils/projects/forms";
import BaseSelect from "@/components/BaseSelect.vue";
import DatePickerField from "@/components/DatePicker/DatePickerField.vue";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import FormMemberPicker from "@/components/Projects/Form/FormMemberPicker.vue";

const MEMBER_LIMIT = 100;

const props = defineProps({
    field: { type: Object, required: true },
    operator: { type: String, default: "equals" },
    modelValue: { type: String, default: "" },
    workspaceId: { type: String, required: true },
    tableId: { type: String, required: true },
});

const emit = defineEmits(["update:modelValue"]);

const alertStore = useAlertStore();

const statusOptions = ref([]);
const members = ref([]);
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

async function loadStatusOptions() {
    const { data } = await workspaceService.getTableStatusTypes(props.workspaceId, props.tableId);

    statusOptions.value = (data.options || []).map((o) => ({ value: o.id, label: o.name }));
}

async function loadMembers() {
    const { data } = await workspaceService.searchWorkspaceMembers(
        props.workspaceId,
        "",
        MEMBER_LIMIT,
        0,
    );

    members.value = data || [];
}

// A saved value is a row id; the picker shows the row's name.
async function loadPicked() {
    if (!props.modelValue) return;

    const { data } = await workspaceService.getLinkedRecordsLite(
        props.workspaceId,
        linkedTableId(props.field),
        { ids: props.modelValue },
    );
    const row = (data || [])[0];

    picked.value = { id: props.modelValue, name: row?.name ?? props.modelValue };
}

const LOADERS = { status: loadStatusOptions, assignee: loadMembers, linked: loadPicked };

watch(
    widget,
    async (kind) => {
        try {
            await LOADERS[kind]?.();
        } catch (error) {
            alertStore.showError(extractErrorMessage(error));
        }
    },
    { immediate: true },
);
</script>
