<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div v-if="status === 'unlicensed'" class="p-3">
        <LicensePurchase feature="forms" />
    </div>

    <div v-else-if="status === 'loading'" class="flex h-full w-full items-center justify-center">
        <BaseSpinner size="lg" class="text-indigo-600" :label="t('common.label.loading')" />
    </div>

    <div
        v-else-if="status === 'forbidden'"
        class="flex h-full w-full flex-col items-center justify-center gap-y-2 p-6 text-center"
    >
        <LockClosedIcon class="h-8 w-8 text-gray-400" aria-hidden="true" />
        <p class="text-sm text-gray-500">{{ t("projects.form_view.no_access") }}</p>
    </div>

    <div
        v-else-if="status === 'failed'"
        class="flex h-full w-full flex-col items-center justify-center gap-y-3 p-6 text-center"
    >
        <p class="text-sm text-gray-500">{{ t("common.error.something_went_wrong") }}</p>
        <BaseButton size="small" variant="secondary" @click="load">
            {{ t("common.button.try_again") }}
        </BaseButton>
    </div>

    <div v-else class="flex h-full w-full flex-col overflow-hidden bg-white">
        <div class="flex shrink-0 items-center gap-x-4 border-b border-gray-200 px-4 py-2">
            <span class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900">
                {{ formData.name || t("projects.form_view.new_form") }}
            </span>
            <nav v-if="formData.id" class="flex gap-x-1" :aria-label="t('projects.form_view.tabs')">
                <button
                    v-for="tab in TABS"
                    :key="tab"
                    type="button"
                    :aria-current="activeTab === tab ? 'page' : undefined"
                    :class="[
                        activeTab === tab
                            ? 'bg-indigo-50 text-indigo-700'
                            : 'text-gray-500 hover:text-gray-700',
                        'rounded-md px-3 py-1.5 text-sm font-medium',
                    ]"
                    @click="activeTab = tab"
                >
                    {{ t(`projects.form_view.tab.${tab}`) }}
                </button>
            </nav>
            <template v-if="formData.id">
                <BaseButton
                    v-if="formData.public_token"
                    size="small"
                    variant="secondary"
                    :prepend-icon="LinkIcon"
                    @click="shareOpen = true"
                >
                    {{ t("projects.form_view.action.share") }}
                </BaseButton>
                <BaseButton
                    v-if="canUpdateForm"
                    size="small"
                    :is-loading="saving"
                    @click="updateForm"
                >
                    {{ t("projects.form_view.action.update") }}
                </BaseButton>
                <BaseButton
                    v-if="canCreateForm"
                    size="small"
                    variant="secondary"
                    :is-disabled="saving"
                    @click="dialogMode = 'saveAsNew'"
                >
                    {{ t("projects.form_view.action.save_as_new") }}
                </BaseButton>
            </template>
            <BaseButton
                v-else-if="canCreateForm"
                size="small"
                :is-disabled="saving"
                @click="dialogMode = 'create'"
            >
                {{ t("projects.form_view.action.save") }}
            </BaseButton>
        </div>

        <FormViewBuild
            v-if="activeTab === 'build'"
            :form="formData"
            :fields="formFields"
            :headers="headers"
            @change-settings="updateSettings"
            @change-field="updateField"
            @reorder="reorderFields"
        />
        <div
            v-else-if="activeTab === 'preview'"
            class="min-h-0 flex-1 overflow-y-auto bg-gray-50 p-6"
        >
            <div class="mx-auto max-w-2xl">
                <FormFill :token="formData.public_token" />
            </div>
        </div>

        <FormViewShareDialog
            v-if="formData.public_token"
            :open="shareOpen"
            :token="formData.public_token"
            @close="shareOpen = false"
        />

        <FormViewNameDialog
            :open="dialogMode !== null"
            :save-as-new="dialogMode === 'saveAsNew'"
            :loading="saving"
            @confirm="createWithName"
            @close="dialogMode = null"
        />
    </div>
</template>

<script setup>
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { LockClosedIcon } from "@heroicons/vue/24/outline";
import { LinkIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import { useFormBuilder } from "@/composables/projects/useFormBuilder";
import { useWorkspaceRolesStore } from "@/store/workspaceRoles";
import BaseButton from "@/components/BaseButton.vue";
import BaseSpinner from "@/components/BaseSpinner.vue";
import LicensePurchase from "@/components/Settings/LicensePurchase.vue";
import FormFill from "@/components/Projects/Form/FormFill.vue";
import FormViewBuild from "@/components/Projects/Form/FormViewBuild.vue";
import FormViewNameDialog from "@/components/Projects/Form/FormViewNameDialog.vue";
import FormViewShareDialog from "@/components/Projects/Form/FormViewShareDialog.vue";

const TABS = ["build", "preview"];

const route = useRoute();
const rolesStore = useWorkspaceRolesStore();
const {
    formData,
    formFields,
    headers,
    status,
    saving,
    load,
    updateSettings,
    updateField,
    reorderFields,
    updateForm,
    createForm,
} = useFormBuilder();

// The server also lets a role that manages this table's views build its
// forms, which only a per-table role can grant.
const managesTableViews = computed(
    () =>
        rolesStore.perTableMode && rolesStore.hasTablePermission(route.params.tid, "manage_views"),
);

const canCreateForm = computed(
    () => rolesStore.hasPermissionToCreateWorkspaceView || managesTableViews.value,
);

const canUpdateForm = computed(
    () => rolesStore.hasPermissionToUpdateWorkspaceView || managesTableViews.value,
);

const dialogMode = ref(null);
const activeTab = ref("build");
const shareOpen = ref(false);

async function createWithName(name) {
    if (await createForm(name)) dialogMode.value = null;
}

watch(
    () => formData.value.id,
    () => {
        activeTab.value = "build";
    },
);
</script>
