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

        <ToggleSwitch
            :model-value="form.active"
            :label="t('projects.form_view.settings.active')"
            @update:model-value="emit('update', { active: $event })"
        />

        <div>
            <label :for="`${id}-accent`" class="block text-sm font-medium text-gray-900">
                {{ t("projects.form_view.settings.accent_color") }}
            </label>
            <input
                :id="`${id}-accent`"
                :value="form.accent_color"
                type="color"
                class="mt-1 h-8 w-full cursor-pointer rounded border border-gray-200 p-0.5"
                @input="emit('update', { accent_color: $event.target.value })"
            />
        </div>

        <FormViewImageInput
            :label="t('projects.form_view.settings.cover_image')"
            :alt="t('projects.form_view.image_alt.cover')"
            :src="formImageUrl(form.workspace_id, form.cover_image)"
            :loading="busy === 'cover'"
            :disabled="!form.id || busy !== ''"
            @pick="setImage('cover', $event)"
            @remove="removeImage('cover')"
        />

        <FormViewImageInput
            :label="t('projects.form_view.settings.logo')"
            :alt="t('projects.form_view.image_alt.logo')"
            :src="formImageUrl(form.workspace_id, form.logo)"
            contain
            :loading="busy === 'logo'"
            :disabled="!form.id || busy !== ''"
            @pick="setImage('logo', $event)"
            @remove="removeImage('logo')"
        />

        <p v-if="!form.id" class="text-xs text-gray-400">
            {{ t("projects.form_view.settings.images_after_save") }}
        </p>
    </aside>
</template>

<script setup>
import { useId } from "vue";
import { t } from "@/i18n/index.js";
import { useFormImages } from "@/composables/projects/useFormImages";
import { formImageUrl } from "@/utils/projects/forms";
import ToggleSwitch from "@/components/ToggleSwitch.vue";
import FormViewImageInput from "@/components/Projects/Form/FormViewImageInput.vue";

const props = defineProps({
    form: {
        type: Object,
        required: true,
    },
});

const emit = defineEmits(["update"]);

const { busy, setImage, removeImage } = useFormImages(
    () => props.form,
    (patch) => emit("update", patch),
);

const id = useId();
</script>
