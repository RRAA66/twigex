<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <FormDialog
        :open="open"
        :title="t('projects.form_view.share.title')"
        :confirm-label="t('projects.form_view.action.copy')"
        @confirm="copyLink"
        @close="emit('close')"
    >
        <p class="text-sm text-gray-500">{{ t("projects.form_view.share.desc") }}</p>
        <input
            :value="link"
            :aria-label="t('projects.form_view.share.title')"
            readonly
            class="mt-3 block w-full rounded-md border-0 py-1.5 text-sm text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-inset focus:ring-indigo-600"
            @focus="$event.target.select()"
        />
    </FormDialog>
</template>

<script setup>
import { computed } from "vue";
import { t } from "@/i18n/index.js";
import { useAlertStore } from "@/store/alerts";
import FormDialog from "@/components/FormDialog.vue";

const props = defineProps({
    open: { type: Boolean, default: false },
    token: { type: String, required: true },
});

const emit = defineEmits(["close"]);

const alertStore = useAlertStore();

const link = computed(() => `${window.location.origin}/forms/public/${props.token}`);

async function copyLink() {
    try {
        await navigator.clipboard.writeText(link.value);
        alertStore.showSuccess(t.value("projects.form_view.success.link_copied"));
        emit("close");
    } catch {
        alertStore.showError(t.value("projects.form_view.error.copy_failed"));
    }
}
</script>
