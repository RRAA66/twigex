<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <aside class="w-60 shrink-0 space-y-4 overflow-y-auto border-l border-gray-200 bg-white p-4">
        <h3
            class="border-b border-gray-100 pb-2 text-xs font-semibold uppercase tracking-wide text-gray-500"
        >
            {{ t("projects.form_view.settings.title") }}
        </h3>

        <ToggleSwitch v-model="form.active" :label="t('projects.form_view.settings.active')" />

        <div>
            <ToggleSwitch
                :model-value="true"
                :label="t('projects.form_view.settings.requires_login')"
                disabled
            />
            <p class="mt-1 text-xs text-gray-400">
                {{ t("projects.form_view.settings.requires_login_disabled_note") }}
            </p>
        </div>

        <div>
            <label :for="`${id}-accent`" class="block text-sm font-medium text-gray-900">
                {{ t("projects.form_view.settings.accent_color") }}
            </label>
            <input
                :id="`${id}-accent`"
                v-model="form.accent_color"
                type="color"
                class="mt-1 h-8 w-full cursor-pointer rounded border border-gray-200 p-0.5"
            />
        </div>
    </aside>
</template>

<script setup>
import { useId } from "vue";
import { t } from "@/i18n/index.js";
import ToggleSwitch from "@/components/ToggleSwitch.vue";

defineProps({
    form: {
        type: Object,
        required: true,
    },
});

const id = useId();
</script>
