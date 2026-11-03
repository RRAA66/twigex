<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <BaseSelect
        :model-value="modelValue"
        :options="options"
        :placeholder="placeholder"
        @update:model-value="emit('update:modelValue', $event)"
    >
        <template #selected="{ option }">
            <span v-if="option?.member" class="flex items-center gap-x-2">
                <span class="h-5 w-5 shrink-0"><UserAvatar :user="option.member" /></span>
                <span class="truncate">{{ option.label }}</span>
            </span>
            <template v-else>{{ placeholder }}</template>
        </template>
        <template #option="{ option, selected }">
            <span class="flex items-center gap-x-2">
                <span v-if="option.member" class="h-5 w-5 shrink-0">
                    <UserAvatar :user="option.member" />
                </span>
                <span :class="[selected ? 'font-semibold' : '', 'truncate']">
                    {{ option.label }}
                </span>
            </span>
        </template>
    </BaseSelect>
</template>

<script setup>
import { computed } from "vue";
import BaseSelect from "@/components/BaseSelect.vue";
import UserAvatar from "@/components/UserAvatar.vue";

const props = defineProps({
    modelValue: { type: String, default: "" },
    members: { type: Array, default: () => [] },
    placeholder: { type: String, default: "" },
});

const emit = defineEmits(["update:modelValue"]);

// The empty first option lets an answer be taken back.
const options = computed(() => [
    { value: "", label: props.placeholder },
    ...props.members.map((m) => ({
        value: m.id,
        label: `${m.name} ${m.lastname}`.trim(),
        member: m,
    })),
]);
</script>
