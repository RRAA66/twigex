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

    <div v-else-if="status === 'forbidden'" class="flex h-full w-full flex-col overflow-hidden">
        <div
            :class="[
                'flex shrink-0 flex-col items-center gap-y-2 p-6 text-center',
                canSeeSubmissions ? 'border-b border-gray-200 bg-white' : 'flex-1 justify-center',
            ]"
        >
            <LockClosedIcon class="h-8 w-8 text-gray-400" aria-hidden="true" />
            <p class="text-sm text-gray-500">{{ t("projects.form_view.no_access") }}</p>
            <p class="max-w-md text-sm text-gray-500">
                {{
                    canSeeSubmissions
                        ? t("projects.form_view.viewer.fill_and_see")
                        : t("projects.form_view.viewer.ask_for_link")
                }}
            </p>
            <RouterLink
                v-if="viewerToken"
                :to="{ name: 'public-form', params: { token: viewerToken } }"
                target="_blank"
                class="text-sm font-medium text-indigo-600 hover:text-indigo-500"
            >
                {{ t("projects.form_view.viewer.fill_in") }}
            </RouterLink>
        </div>
        <FormViewSubmissions
            v-if="canSeeSubmissions"
            :workspace-id="route.params.id"
            :table-id="route.params.tid"
            :form-id="route.params.fid"
            @loaded="viewerToken = $event.public_token || ''"
        />
    </div>

    <div
        v-else-if="status === 'missing'"
        class="flex h-full w-full flex-col items-center justify-center gap-y-2 p-6 text-center"
    >
        <p class="text-sm text-gray-500">{{ t("projects.form_view.not_found") }}</p>
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
            <nav class="flex gap-x-1" :aria-label="t('projects.form_view.tabs')">
                <button
                    v-for="tab in tabs"
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
                <BaseButton v-if="canUpdateForm" size="small" :is-loading="saving" @click="update">
                    {{ t("projects.form_view.action.update") }}
                </BaseButton>
                <BaseButton
                    v-if="canCreateForm"
                    size="small"
                    variant="secondary"
                    :is-disabled="saving"
                    @click="openNameDialog('saveAsNew')"
                >
                    {{ t("projects.form_view.action.save_as_new") }}
                </BaseButton>
            </template>
            <BaseButton
                v-else-if="canCreateForm"
                size="small"
                :is-disabled="saving"
                @click="openNameDialog('create')"
            >
                {{ t("projects.form_view.action.save") }}
            </BaseButton>
        </div>

        <div
            v-if="changedElsewhere"
            role="alert"
            class="flex shrink-0 items-center gap-x-3 border-b border-yellow-200 bg-yellow-50 px-4 py-2"
        >
            <p class="flex-1 text-sm text-yellow-800">
                {{ t("projects.form_view.changed_elsewhere") }}
            </p>
            <BaseButton size="small" variant="secondary" @click="load">
                {{ t("projects.form_view.action.reload") }}
            </BaseButton>
        </div>

        <FormViewBuild
            v-if="activeTab === 'build'"
            :form="formData"
            :fields="formFields"
            :headers="headers"
            :incomplete-fields="incompleteFields"
            :cover-src="imageSrc('cover')"
            :logo-src="imageSrc('logo')"
            @change-settings="updateSettings"
            @change-field="updateField"
            @reorder="reorderFields"
            @pick-image="pickImage"
            @remove-image="removeImage"
        />
        <div
            v-else-if="activeTab === 'preview'"
            class="min-h-0 flex-1 overflow-y-auto bg-gray-50 p-6"
        >
            <div class="mx-auto max-w-2xl">
                <FormFill :draft="previewForm" preview />
            </div>
        </div>
        <FormViewSubmissions
            v-else
            :workspace-id="formData.workspace_id"
            :table-id="formData.table_id"
            :form-id="formData.id"
            :fields="formFields"
        />

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

        <ConfirmDialog
            :open="!!answerLeave"
            :title="t('projects.form_view.unsaved_title')"
            :message="t('projects.form_view.unsaved_message')"
            :confirm-label="t('projects.form_view.unsaved_leave')"
            @confirm="answerLeave(true)"
            @close="answerLeave(false)"
        />
    </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { onBeforeRouteLeave, onBeforeRouteUpdate, RouterLink, useRoute } from "vue-router";
import { LockClosedIcon } from "@heroicons/vue/24/outline";
import { LinkIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import { useFormBuilder } from "@/composables/projects/useFormBuilder";
import { useWorkspaceRolesStore } from "@/store/workspaceRoles";
import BaseButton from "@/components/BaseButton.vue";
import BaseSpinner from "@/components/BaseSpinner.vue";
import ConfirmDialog from "@/components/ConfirmDialog.vue";
import LicensePurchase from "@/components/Settings/LicensePurchase.vue";
import FormFill from "@/components/Projects/Form/FormFill.vue";
import FormViewBuild from "@/components/Projects/Form/FormViewBuild.vue";
import FormViewNameDialog from "@/components/Projects/Form/FormViewNameDialog.vue";
import FormViewShareDialog from "@/components/Projects/Form/FormViewShareDialog.vue";
import FormViewSubmissions from "@/components/Projects/Form/FormViewSubmissions.vue";

const route = useRoute();
const rolesStore = useWorkspaceRolesStore();
const {
    formData,
    formFields,
    headers,
    status,
    saving,
    incompleteFields,
    changedElsewhere,
    isDirty,
    load,
    checkConditions,
    updateSettings,
    updateField,
    reorderFields,
    updateForm,
    createForm,
    imageSrc,
    pickImage,
    removeImage,
} = useFormBuilder();

// A new form has no submissions to list yet.
const tabs = computed(() =>
    formData.value.id ? ["build", "preview", "submissions"] : ["build", "preview"],
);

// Preview draws the form as it is being built, saved or not, as its link will
// serve it once saved.
const previewForm = computed(() => ({
    ...formData.value,
    fields: formFields.value,
    can_submit: true,
    cover_src: imageSrc("cover"),
    logo_src: imageSrc("logo"),
}));

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

// Who may not build the form may still fill it in, and see what was sent
// with a role that opens both forms and grids, as the server allows.
const canSeeSubmissions = computed(
    () =>
        Boolean(route.params.fid) &&
        rolesStore.hasPermission("views_form") &&
        rolesStore.hasPermission("views_grid"),
);

const viewerToken = ref("");
const dialogMode = ref(null);
const activeTab = ref("build");
const shareOpen = ref(false);
const answerLeave = ref(null);

// An incomplete condition is flagged on the build tab, so that is shown.
function conditionsReady() {
    if (checkConditions()) return true;

    activeTab.value = "build";

    return false;
}

function update() {
    if (conditionsReady()) updateForm();
}

function openNameDialog(mode) {
    if (conditionsReady()) dialogMode.value = mode;
}

async function createWithName(name) {
    if (await createForm(name)) dialogMode.value = null;
}

watch(
    () => formData.value.id,
    () => {
        activeTab.value = "build";
    },
);
function confirmLeave() {
    if (!isDirty.value) return true;

    return new Promise((resolve) => {
        answerLeave.value = (leave) => {
            answerLeave.value = null;
            resolve(leave);
        };
    });
}

function warnBeforeUnload(event) {
    if (!isDirty.value) return;
    event.preventDefault();
    event.returnValue = "";
}

onBeforeRouteLeave(confirmLeave);

onBeforeRouteUpdate((to, from) =>
    to.params.id === from.params.id &&
    to.params.tid === from.params.tid &&
    to.params.fid === from.params.fid
        ? true
        : confirmLeave(),
);

onMounted(() => window.addEventListener("beforeunload", warnBeforeUnload));

onBeforeUnmount(() => window.removeEventListener("beforeunload", warnBeforeUnload));
</script>
