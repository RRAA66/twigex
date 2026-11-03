<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="min-h-0 flex-1 overflow-y-auto bg-gray-50 p-6">
        <DataTable :columns="columns" :items="submissions" :loading="loading">
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

            <template #task_id="{ item }">
                <span class="font-mono text-xs text-gray-500">{{ item.task_id }}</span>
            </template>
        </DataTable>
    </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { t } from "@/i18n/index.js";
import useDateOperations from "@/composables/useDateOperations.js";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import DataTable from "@/components/DataTable.vue";
import UserAvatar from "@/components/UserAvatar.vue";

const props = defineProps({
    workspaceId: { type: String, required: true },
    tableId: { type: String, required: true },
    formId: { type: String, required: true },
});

const alertStore = useAlertStore();
const { getDateAndTime } = useDateOperations();

const submissions = ref([]);
const loading = ref(true);

const columns = computed(() => [
    { key: "submitted_by", label: t.value("projects.form_view.submissions.submitted_by") },
    { key: "submitted_at", label: t.value("projects.form_view.submissions.date") },
    {
        key: "task_id",
        label: t.value("projects.form_view.submissions.task_id"),
        hiddenBelow: "sm",
    },
]);

function submitterName(submission) {
    const info = submission.submitter_info;
    const name = info ? `${info.name} ${info.lastname}`.trim() : "";

    return name || t.value("common.label.unknown_user");
}

async function load() {
    try {
        const { data } = await formService.getSubmissions(
            props.workspaceId,
            props.tableId,
            props.formId,
        );

        submissions.value = data;
    } catch (error) {
        alertStore.showError(extractErrorMessage(error));
    } finally {
        loading.value = false;
    }
}

onMounted(load);
</script>
