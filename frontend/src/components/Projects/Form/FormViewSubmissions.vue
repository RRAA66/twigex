<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="min-h-0 flex-1 overflow-y-auto bg-gray-50 p-6">
        <DataTable
            :columns="columns"
            :items="submissions"
            :loading="loading"
            :total-items="total"
            :current-page="page"
            :items-per-page="PAGE_SIZE"
            clickable
            @row-click="open"
            @update:current-page="load"
        >
            <template #empty>
                {{ t("projects.form_view.no_submissions") }}
            </template>
            <template #submitted_by="{ item }">
                <div class="flex min-w-0 items-center gap-x-3">
                    <span class="size-8 shrink-0">
                        <UserAvatar :user="item.submitter_info" :user-id="item.submitted_by" />
                    </span>
                    <span class="truncate text-gray-900">{{ submitterName(item) }}</span>
                </div>
            </template>
            <template #submitted_at="{ item }">
                {{ getDateAndTime(item.submitted_at) }}
            </template>
            <template v-for="(field, index) in shownFields" #[field.header_name]="{ item }">
                <span
                    v-if="!item.row && index === 0"
                    :key="field.header_name"
                    class="italic text-gray-400"
                >
                    {{ t("projects.form_view.submissions.row_unavailable") }}
                </span>
                <FormSubmissionValue
                    v-else-if="item.row"
                    :key="field.header_name"
                    :field="field"
                    :value="item.row[field.header_name]"
                />
            </template>
        </DataTable>
    </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { t } from "@/i18n/index.js";
import useDateOperations from "@/composables/useDateOperations.js";
import { useOpenTask } from "@/composables/projects/useOpenTask";
import { onProjectChange } from "@/js/websocket";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import { fillableFormFields } from "@/utils/projects/forms";
import DataTable from "@/components/DataTable.vue";
import UserAvatar from "@/components/UserAvatar.vue";
import FormSubmissionValue from "@/components/Projects/Form/FormSubmissionValue.vue";

const PAGE_SIZE = 50;
const RELOAD_DELAY_MS = 500;

const props = defineProps({
    workspaceId: { type: String, required: true },
    tableId: { type: String, required: true },
    formId: { type: String, required: true },
    fields: { type: Array, default: () => [] },
});

const alertStore = useAlertStore();
const { getDateAndTime } = useDateOperations();
const { openTask } = useOpenTask();

const submissions = ref([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);
let request = 0;
let reloadTimer = null;

const shownFields = computed(() => fillableFormFields(props.fields));

const columns = computed(() => [
    { key: "submitted_by", label: t.value("projects.form_view.submissions.submitted_by") },
    { key: "submitted_at", label: t.value("projects.form_view.submissions.date") },
    ...shownFields.value.map((field, index) => ({
        key: field.header_name,
        label: field.label || field.display_name || field.header_name,
        hiddenBelow: index > 0 ? "md" : undefined,
    })),
]);

function submitterName(submission) {
    const info = submission.submitter_info;
    const name = info ? `${info.name} ${info.lastname}`.trim() : "";

    return name || t.value("common.label.unknown_user");
}

async function load(next = page.value) {
    const current = ++request;

    loading.value = true;
    try {
        const { data } = await formService.getSubmissions(
            props.workspaceId,
            props.tableId,
            props.formId,
            next,
        );

        if (current !== request) return;

        submissions.value = data.items;
        total.value = data.total;
        page.value = next;
    } catch (error) {
        if (current === request) alertStore.showError(extractErrorMessage(error));
    } finally {
        if (current === request) loading.value = false;
    }
}

function open(submission) {
    if (!submission.row) return;

    openTask({
        workspaceId: props.workspaceId,
        tableId: props.tableId,
        taskId: submission.task_id,
    });
}

// A new submission, or an edit to a row one made, shows without a reload.
const stopChanges = onProjectChange((change) => {
    if (change?.table_id !== props.tableId || !String(change.type).startsWith("task_")) return;

    clearTimeout(reloadTimer);
    reloadTimer = setTimeout(() => load(), RELOAD_DELAY_MS);
});

onMounted(() => load(1));

onBeforeUnmount(() => {
    stopChanges();
    clearTimeout(reloadTimer);
});
</script>
