<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="space-y-2">
        <BaseCheckbox
            :model-value="Boolean(rule)"
            :label="t('projects.form_view.field.conditional_label')"
            :description="rule ? t('projects.form_view.field.conditional_hint') : ''"
            @update:model-value="setEnabled"
        />
        <div v-if="rule" class="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <BaseSelect
                :model-value="rule.trigger_field"
                :label="t('projects.form_view.field.condition_field_label')"
                :options="triggerOptions"
                :placeholder="t('projects.form_view.field.condition_field_placeholder')"
                @update:model-value="setTrigger"
            />
            <BaseSelect
                :model-value="rule.operator"
                :label="t('projects.form_view.field.condition_operator_label')"
                :options="operatorOptions"
                @update:model-value="setOperator"
            />
            <FormConditionValueInput
                v-if="trigger"
                :key="rule.trigger_field"
                :model-value="rule.value"
                :label="t('projects.form_view.field.condition_value_label')"
                :field="trigger"
                :operator="rule.operator"
                :workspace-id="workspaceId"
                :table-id="tableId"
                @update:model-value="change({ value: $event })"
            />
        </div>
        <p v-if="incomplete" class="text-sm text-red-600">
            {{ t("projects.form_view.field.condition_incomplete") }}
        </p>
    </div>
</template>

<script setup>
import { computed } from "vue";
import { t } from "@/i18n/index.js";
import {
    CONDITION_OPERATORS,
    conditionTriggers,
    conditionValueWidget,
} from "@/utils/projects/forms";
import BaseCheckbox from "@/components/BaseCheckbox.vue";
import BaseSelect from "@/components/BaseSelect.vue";
import FormConditionValueInput from "@/components/Projects/Form/FormConditionValueInput.vue";

const props = defineProps({
    field: { type: Object, required: true },
    fields: { type: Array, default: () => [] },
    workspaceId: { type: String, required: true },
    tableId: { type: String, required: true },
    incomplete: { type: Boolean, default: false },
});

const emit = defineEmits(["update"]);

const rule = computed(() => props.field.visibility_rule);

function setEnabled(on) {
    emit("update", on ? { trigger_field: "", operator: "equals", value: "" } : null);
}

function change(patch) {
    emit("update", { ...rule.value, ...patch });
}

const trigger = computed(() =>
    props.fields.find((f) => f.header_name === rule.value?.trigger_field),
);

const triggerOptions = computed(() =>
    conditionTriggers(props.fields, props.field).map((f) => ({
        value: f.header_name,
        label: f.label || f.display_name || f.header_name,
    })),
);

const operatorOptions = computed(() =>
    CONDITION_OPERATORS.map((op) => ({
        value: op,
        label: t.value(`projects.form_view.field.condition_${op}`),
    })),
);

function setTrigger(name) {
    change({ trigger_field: name, value: "" });
}

// A value picked with one input, such as a row id, means nothing to another.
function setOperator(operator) {
    const field = trigger.value || {};
    const kept =
        conditionValueWidget(field, operator) === conditionValueWidget(field, rule.value.operator);

    change({ operator, value: kept ? rule.value.value : "" });
}
</script>
