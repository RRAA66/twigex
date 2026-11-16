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
            <template #toolbar>
                <div class="flex flex-wrap items-end gap-3">
                    <div class="w-44">
                        <label
                            :for="`${id}-from`"
                            class="mb-1 block text-xs font-medium text-gray-600"
                        >
                            {{ t("projects.form_view.submissions.from") }}
                        </label>
                        <DatePickerField
                            :id="`${id}-from`"
                            v-model="fromDay"
                            :placeholder="t('projects.form_view.placeholder.pick_date')"
                        />
                    </div>
                    <div class="w-44">
                        <label
                            :for="`${id}-to`"
                            class="mb-1 block text-xs font-medium text-gray-600"
                        >
                            {{ t("projects.form_view.submissions.to") }}
                        </label>
                        <DatePickerField
                            :id="`${id}-to`"
                            v-model="toDay"
                            :placeholder="t('projects.form_view.placeholder.pick_date')"
                        />
                    </div>
                    <p v-if="rangeReversed" role="alert" class="pb-2 text-xs text-red-600">
                        {{ t("projects.form_view.submissions.range_reversed") }}
                    </p>
                    <BaseButton
                        type="button"
                        size="small"
                        variant="secondary"
                        class="ml-auto"
                        :prepend-icon="TableCellsIcon"
                        @click="downloadCSV"
                    >
                        {{ t("projects.form_view.submissions.export") }}
                    </BaseButton>
                </div>
            </template>
            <template #empty>
                {{ emptyText }}
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
import { computed, onBeforeUnmount, onMounted, ref, useId, watch } from "vue";
import { TableCellsIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import useDateOperations from "@/composables/useDateOperations.js";
import { useOpenTask } from "@/composables/projects/useOpenTask";
import { onProjectChange } from "@/js/websocket";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";
import { useAlertStore } from "@/store/alerts";
import { useUserStore } from "@/store/user";
import { useWorkspaceStore } from "@/store/workspaces";
import { extractErrorMessage } from "@/utils/errors";
import { fillableFormFields, formFieldType } from "@/utils/projects/forms";
import { tableDisplayNames } from "@/utils/projects/gridView";
import BaseButton from "@/components/BaseButton.vue";
import DataTable from "@/components/DataTable.vue";
import DatePickerField from "@/components/DatePicker/DatePickerField.vue";
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
const { getDateAndTime, getTimestampFromDateString, getEndOfDayTimestamp } = useDateOperations();
const id = useId();
const { openTask } = useOpenTask();

const submissions = ref([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);
const fromDay = ref("");
const toDay = ref("");
let request = 0;
let reloadTimer = null;

// The days picked, in the user's timezone, as the server's unix range: from
// the start of the first day to the end of the last.
const range = computed(() => {
    const params = {};

    if (fromDay.value) params.from = getTimestampFromDateString(fromDay.value);
    if (toDay.value) params.to = getEndOfDayTimestamp(toDay.value) + 1;

    return params;
});

const rangeReversed = computed(
    () =>
        range.value.from !== undefined &&
        range.value.to !== undefined &&
        range.value.from >= range.value.to,
);

const emptyText = computed(() =>
    fromDay.value || toDay.value
        ? t.value("projects.form_view.submissions.none_in_range")
        : t.value("projects.form_view.no_submissions"),
);

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

    if (rangeReversed.value) {
        submissions.value = [];
        total.value = 0;
        page.value = 1;
        loading.value = false;

        return;
    }

    loading.value = true;
    try {
        const { data } = await formService.getSubmissions(
            props.workspaceId,
            props.tableId,
            props.formId,
            { page: next, ...range.value },
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

// A master link names a table, and only the grid and board load the names
// the task panel and this list show it by.
async function loadTableNames() {
    if (!shownFields.value.some((f) => formFieldType(f) === "master-link")) return;

    try {
        const { data } = await workspaceService.getWorkspaceTables(props.workspaceId, "nav");

        useWorkspaceStore().setTableNameLookup(tableDisplayNames(data || []));
    } catch (error) {
        alertStore.showError(extractErrorMessage(error));
    }
}

// The server streams the file and the browser saves it straight to disk, as
// the task report's export does.
function downloadCSV() {
    const userStore = useUserStore();
    const params = new URLSearchParams(range.value);

    if (userStore.getTimezone) params.set("tz", userStore.getTimezone);
    params.set("clock", userStore.getClockDisplay);
    params.set(
        "labels",
        JSON.stringify({
            submitted_by: t.value("projects.form_view.submissions.submitted_by"),
            submitted_at: t.value("projects.form_view.submissions.date"),
        }),
    );

    const a = document.createElement("a");

    a.href = `/api/workspaces/${props.workspaceId}/table/${props.tableId}/forms/${props.formId}/submissions/export?${params}`;
    a.download = "";
    document.body.appendChild(a);
    a.click();
    a.remove();
}

function open(submission) {
    if (!submission.row) return;

    openTask({
        workspaceId: props.workspaceId,
        tableId: props.tableId,
        taskId: submission.task_id,
    });
}

const stopChanges = onProjectChange((change) => {
    if (change?.table_id !== props.tableId || !String(change.type).startsWith("task_")) return;

    clearTimeout(reloadTimer);
    reloadTimer = setTimeout(() => load(), RELOAD_DELAY_MS);
});

watch([fromDay, toDay], () => load(1));

onMounted(() => {
    load(1);
    loadTableNames();
});

onBeforeUnmount(() => {
    stopChanges();
    clearTimeout(reloadTimer);
});
</script>
