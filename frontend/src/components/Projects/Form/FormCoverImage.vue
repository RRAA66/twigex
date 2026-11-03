<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div
        ref="frame"
        :class="[
            repositioning
                ? 'cursor-grab touch-none focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-indigo-600 active:cursor-grabbing'
                : '',
            'relative h-40 overflow-hidden bg-gray-100',
        ]"
        v-bind="repositioning ? sliderAttrs : {}"
        @pointerdown="startDrag"
        @pointermove="drag"
        @pointerup="endDrag"
        @pointercancel="endDrag"
        @keydown="moveByKey"
    >
        <img
            :src="src"
            :alt="t('projects.form_view.image_alt.cover')"
            draggable="false"
            class="h-full w-full select-none object-cover"
            :style="{ objectPosition: `50% ${position}%` }"
        />
        <p v-if="repositionable" :id="hintId" class="sr-only">
            {{ t("projects.form_view.cover_position_hint") }}
        </p>
        <div
            v-if="repositionable"
            ref="toggle"
            class="absolute bottom-2 right-2"
            @pointerdown.stop
            @keydown.stop
        >
            <BaseButton
                size="small"
                variant="secondary"
                :prepend-icon="repositioning ? CheckIcon : ArrowsUpDownIcon"
                @click="toggleRepositioning"
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
import { computed, nextTick, ref, useId } from "vue";
import { ArrowsUpDownIcon, CheckIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import BaseButton from "@/components/BaseButton.vue";

const KEY_STEP = 5;

const props = defineProps({
    src: { type: String, required: true },
    position: { type: Number, default: 50 },
    repositionable: { type: Boolean, default: false },
});

const emit = defineEmits(["update:position"]);

const hintId = useId();

const frame = ref(null);
const toggle = ref(null);
const repositioning = ref(false);
const dragStart = ref(null);

const sliderAttrs = computed(() => ({
    tabindex: 0,
    role: "slider",
    "aria-label": t.value("projects.form_view.cover_position"),
    "aria-describedby": hintId,
    "aria-orientation": "vertical",
    "aria-valuemin": 0,
    "aria-valuemax": 100,
    "aria-valuenow": props.position,
}));

async function toggleRepositioning() {
    repositioning.value = !repositioning.value;
    if (!repositioning.value) return;

    await nextTick();
    frame.value.focus();
}

async function finish() {
    repositioning.value = false;
    dragStart.value = null;

    await nextTick();
    toggle.value.querySelector("button")?.focus();
}

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

    emit("update:position", clamp(dragStart.value.position - moved));
}

function endDrag() {
    dragStart.value = null;
}

// The arrows move the image as dragging it would: up shows more of its
// bottom.
function moveByKey(event) {
    if (!repositioning.value) return;

    switch (event.key) {
        case "ArrowUp":
            emit("update:position", clamp(props.position + KEY_STEP));
            break;
        case "ArrowDown":
            emit("update:position", clamp(props.position - KEY_STEP));
            break;
        case "Enter":
        case "Escape":
            finish();
            break;
        default:
            return;
    }

    event.preventDefault();
}

function clamp(position) {
    return Math.round(Math.min(100, Math.max(0, position)));
}
</script>
