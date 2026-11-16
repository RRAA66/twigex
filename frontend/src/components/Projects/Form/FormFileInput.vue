<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div>
        <ul
            v-if="files.length > 0"
            class="mb-2 divide-y divide-gray-100 rounded-md bg-white ring-1 ring-gray-200"
        >
            <li
                v-for="file in files"
                :key="file.id"
                class="flex items-center gap-x-2 px-3 py-2 text-sm"
            >
                <PaperClipIcon class="h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
                <span class="min-w-0 flex-1 truncate text-gray-700">{{ file.name }}</span>
                <button
                    type="button"
                    class="rounded p-0.5 text-gray-400 hover:text-gray-600"
                    :aria-label="t('projects.form_view.action.remove_file', { name: file.name })"
                    @click="remove(file.id)"
                >
                    <XMarkIcon class="h-4 w-4" aria-hidden="true" />
                </button>
            </li>
        </ul>

        <BaseButton
            :id="id"
            type="button"
            size="small"
            variant="secondary"
            :prepend-icon="ArrowUpTrayIcon"
            :is-loading="uploading"
            :is-disabled="files.length >= MAX_FILES"
            v-bind="$attrs"
            @click="input.click()"
        >
            {{ t("projects.form_view.action.add_files") }}
        </BaseButton>
        <input ref="input" type="file" multiple class="hidden" @change="pick" />
    </div>
</template>

<script setup>
import { computed, ref } from "vue";
import { ArrowUpTrayIcon } from "@heroicons/vue/20/solid";
import { PaperClipIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { t } from "@/i18n/index.js";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import BaseButton from "@/components/BaseButton.vue";

// The server keeps no more files than this for one field.
const MAX_FILES = 10;

// The field's label names the button, which takes its aria attributes.
defineOptions({ inheritAttrs: false });

const props = defineProps({
    id: { type: String, required: true },
    modelValue: { type: Array, default: null },
    // Without one, as in the builder's preview, files stay on the screen.
    token: { type: String, default: "" },
});

const emit = defineEmits(["update:modelValue"]);

const alertStore = useAlertStore();

const input = ref(null);
const uploading = ref(false);

const files = computed(() => props.modelValue || []);

let keptFiles = 0;

async function pick(event) {
    const all = [...event.target.files];
    const picked = all.slice(0, MAX_FILES - files.value.length);

    event.target.value = "";
    if (picked.length < all.length) {
        alertStore.showError(
            t.value("projects.form_view.error.too_many_files", { max: MAX_FILES }),
        );
    }

    if (picked.length === 0) return;

    if (!props.token) {
        const kept = picked.map((f) => ({ id: `kept-${++keptFiles}`, name: f.name, size: f.size }));

        emit("update:modelValue", [...files.value, ...kept]);

        return;
    }

    uploading.value = true;
    try {
        const { data } = await formService.uploadPublicFormFiles(props.token, picked);

        emit("update:modelValue", [...files.value, ...data]);
    } catch (error) {
        alertStore.showError(extractErrorMessage(error));
    } finally {
        uploading.value = false;
    }
}

function remove(fileId) {
    emit(
        "update:modelValue",
        files.value.filter((f) => f.id !== fileId),
    );
}
</script>
