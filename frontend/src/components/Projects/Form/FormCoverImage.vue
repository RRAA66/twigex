<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div
        ref="frame"
        :class="[
            repositioning ? 'cursor-grab touch-none active:cursor-grabbing' : '',
            'relative h-40 overflow-hidden bg-gray-100',
        ]"
        @pointerdown="startDrag"
        @pointermove="drag"
        @pointerup="endDrag"
        @pointercancel="endDrag"
    >
        <img
            :src="src"
            :alt="t('projects.form_view.image_alt.cover')"
            draggable="false"
            class="h-full w-full select-none object-cover"
            :style="{ objectPosition: `50% ${position}%` }"
        />
        <div v-if="repositionable" class="absolute bottom-2 right-2" @pointerdown.stop>
            <BaseButton
                size="small"
                variant="secondary"
                :prepend-icon="repositioning ? CheckIcon : ArrowsUpDownIcon"
                @click="repositioning = !repositioning"
            >
                {{
                    t(
                        repositioning
                            ? "projects.form_view.action.reposition_done"
                            : "projects.form_view.action.reposition",
                    )
                }}
            </BaseButton>
        </div>
    </div>
</template>

<script setup>
import { ref } from "vue";
import { ArrowsUpDownIcon, CheckIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import BaseButton from "@/components/BaseButton.vue";

const props = defineProps({
    src: { type: String, required: true },
    position: { type: Number, default: 50 },
    repositionable: { type: Boolean, default: false },
});

const emit = defineEmits(["update:position"]);

const frame = ref(null);
const repositioning = ref(false);
const dragStart = ref(null);

function startDrag(event) {
    if (!repositioning.value) return;

    event.currentTarget.setPointerCapture?.(event.pointerId);
    dragStart.value = { y: event.clientY, position: props.position };
}

// Dragging the image down shows more of its top, so the position falls.
function drag(event) {
    if (!dragStart.value) return;

    const height = frame.value.clientHeight || 1;
    const moved = ((event.clientY - dragStart.value.y) / height) * 100;
    const position = Math.round(Math.min(100, Math.max(0, dragStart.value.position - moved)));

    emit("update:position", position);
}

function endDrag() {
    dragStart.value = null;
}
</script>
