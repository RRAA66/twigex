<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <span v-if="empty" class="text-gray-400">–</span>

    <span
        v-else-if="type === 'single-select'"
        class="inline-flex max-w-full items-center gap-x-1.5 rounded-full bg-white px-2 py-0.5 text-xs font-medium text-gray-700 ring-1 ring-inset ring-gray-200"
    >
        <span
            v-if="value.color"
            class="h-2 w-2 flex-none rounded-full"
            :class="getStatusStyle(value).class"
            :style="getStatusStyle(value).style"
            aria-hidden="true"
        />
        <span class="truncate">{{ value.name }}</span>
    </span>

    <span v-else-if="type === 'link'" class="flex max-w-xs flex-wrap gap-1">
        <span
            v-for="item in value"
            :key="item.id"
            class="truncate rounded-md bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700"
        >
            {{ item.restricted ? t("projects.form_view.submissions.hidden_item") : item.name }}
        </span>
    </span>

    <span v-else-if="type === 'assignee'" class="flex min-w-0 items-center gap-x-2">
        <span class="size-6 shrink-0"><UserAvatar :user-id="value" /></span>
        <span class="truncate text-gray-900">{{ personName }}</span>
    </span>

    <CheckIcon
        v-else-if="type === 'checkbox'"
        class="h-5 w-5 text-indigo-600"
        :aria-label="t('projects.form_view.placeholder.yes')"
    />

    <span v-else-if="type === 'file'">
        {{ t("projects.form_view.submissions.files", { count: value.length }) }}
    </span>

    <span v-else-if="type === 'date'">{{ day }}</span>

    <span v-else-if="type === 'master-link'" class="block max-w-xs truncate text-gray-900">
        {{ workspaceStore.tableNameOf(value) }}
    </span>

    <span v-else class="block max-w-xs truncate text-gray-900">{{ value }}</span>
</template>

<script setup>
import { computed } from "vue";
import { CheckIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import useDateOperations from "@/composables/useDateOperations.js";
import { useUserState } from "@/composables/useUser";
import { formFieldType } from "@/utils/projects/forms";
import { getStatusStyle } from "@/utils/projects/cells";
import { useWorkspaceStore } from "@/store/workspaces";
import UserAvatar from "@/components/UserAvatar.vue";

const props = defineProps({
    field: { type: Object, required: true },
    value: { type: [String, Number, Boolean, Object, Array], default: null },
});

const workspaceStore = useWorkspaceStore();
const { formatDay, getDate } = useDateOperations();

const type = computed(() => formFieldType(props.field));

// A checkbox is stored as 0 or 1, and an unanswered field as empty.
const empty = computed(() => {
    const v = props.value;

    if (v == null || v === "" || (Array.isArray(v) && v.length === 0)) return true;
    if (type.value === "checkbox") return !(v === true || v === 1 || v === "1");

    return false;
});

const { user } = useUserState(() => (type.value === "assignee" ? props.value || null : null));

const personName = computed(() =>
    [user.value?.name, user.value?.lastname].filter(Boolean).join(" "),
);

const day = computed(() => formatDay(props.value) || getDate(props.value));
</script>
