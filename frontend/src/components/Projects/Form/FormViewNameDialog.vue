<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <FormDialog
        :open="open"
        :title="
            saveAsNew
                ? t('projects.form_view.dialog.save_as_new_title')
                : t('projects.form_view.dialog.create_title')
        "
        :confirm-label="
            saveAsNew
                ? t('projects.form_view.action.save_as_new')
                : t('projects.form_view.dialog.create_btn')
        "
        :confirm-disabled="!name.trim()"
        :loading="loading"
        width="sm:max-w-md"
        @confirm="emit('confirm', name.trim())"
        @close="emit('close')"
    >
        <label :for="`${id}-name`" class="sr-only">
            {{ t("projects.form_view.dialog.placeholder") }}
        </label>
        <input
            :id="`${id}-name`"
            v-model="name"
            type="text"
            class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
            :placeholder="t('projects.form_view.dialog.placeholder')"
        />
    </FormDialog>
</template>

<script setup>
import { ref, useId, watch } from "vue";
import { t } from "@/i18n/index.js";
import FormDialog from "@/components/FormDialog.vue";

const props = defineProps({
    open: {
        type: Boolean,
        default: false,
    },
    saveAsNew: {
        type: Boolean,
        default: false,
    },
    loading: {
        type: Boolean,
        default: false,
    },
});

const emit = defineEmits(["confirm", "close"]);

const id = useId();

const name = ref("");

watch(
    () => props.open,
    (open) => {
        if (open) name.value = "";
    },
);
</script>
