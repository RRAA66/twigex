<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="space-y-2">
        <BaseCheckbox
            v-model="enabled"
            :label="t('projects.form_view.field.conditional_label')"
            :description="rule ? t('projects.form_view.field.conditional_hint') : ''"
        />
        <div v-if="rule" class="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <BaseSelect
                :model-value="rule.trigger_field"
                :options="triggerOptions"
                :placeholder="t('projects.form_view.field.condition_field_placeholder')"
                @update:model-value="setTrigger"
            />
            <BaseSelect
                :model-value="rule.operator"
                :options="operatorOptions"
                @update:model-value="setOperator"
            />
            <FormConditionValueInput
                v-if="trigger"
                :key="rule.trigger_field"
                v-model="rule.value"
                :field="trigger"
                :operator="rule.operator"
                :workspace-id="workspaceId"
                :table-id="tableId"
            />
        </div>
    </div>
</template>

<script setup>
import { computed } from "vue";
import { t } from "@/i18n/index.js";
import { conditionValueWidget } from "@/utils/projects/conditionValueWidget";
import { CONDITION_OPERATORS, conditionTriggers } from "@/utils/projects/forms";
import BaseCheckbox from "@/components/BaseCheckbox.vue";
import BaseSelect from "@/components/BaseSelect.vue";
import FormConditionValueInput from "@/components/Projects/Form/FormConditionValueInput.vue";

const props = defineProps({
    field: { type: Object, required: true },
    fields: { type: Array, default: () => [] },
    workspaceId: { type: String, required: true },
    tableId: { type: String, required: true },
});

const rule = computed(() => props.field.visibility_rule);

const enabled = computed({
    get: () => Boolean(rule.value),
    set: (on) => {
        props.field.visibility_rule = on
            ? { trigger_field: "", operator: "equals", value: "" }
            : null;
    },
});

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
    rule.value.trigger_field = name;
    rule.value.value = "";
}

// A value picked with one input, such as a row id, means nothing to another.
function setOperator(operator) {
    const field = trigger.value || {};

    if (
        conditionValueWidget(field, operator) !== conditionValueWidget(field, rule.value.operator)
    ) {
        rule.value.value = "";
    }

    rule.value.operator = operator;
}
</script>
