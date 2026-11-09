<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div>
        <span class="block text-sm font-medium text-gray-900">{{ label }}</span>
        <img
            v-if="src"
            :src="src"
            :alt="alt"
            :class="[
                contain ? 'object-contain' : 'object-cover',
                'mt-1 h-20 w-full rounded border border-gray-200 bg-gray-50',
            ]"
        />
        <div class="mt-2 flex gap-x-2">
            <BaseButton
                size="small"
                variant="secondary"
                :prepend-icon="PhotoIcon"
                :is-loading="loading"
                :is-disabled="disabled"
                @click="input.click()"
            >
                {{
                    t(
                        src
                            ? "projects.form_view.action.replace_image"
                            : "projects.form_view.action.upload_image",
                    )
                }}
            </BaseButton>
            <BaseButton
                v-if="src"
                size="small"
                variant="secondary"
                :is-disabled="loading"
                @click="emit('remove')"
            >
                {{ t("common.button.remove") }}
            </BaseButton>
        </div>
        <input
            ref="input"
            type="file"
            accept="image/png,image/jpeg,image/gif,image/webp"
            class="hidden"
            :aria-label="label"
            @change="pick"
        />
    </div>
</template>

<script setup>
import { ref } from "vue";
import { PhotoIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import BaseButton from "@/components/BaseButton.vue";

defineProps({
    label: { type: String, required: true },
    alt: { type: String, required: true },
    src: { type: String, default: "" },
    contain: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
});

const emit = defineEmits(["pick", "remove"]);

const input = ref(null);

function pick(event) {
    const [file] = event.target.files;

    event.target.value = "";
    if (file) emit("pick", file);
}
</script>
