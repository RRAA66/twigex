<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <Popover class="relative" @click.stop @pointerdown.stop>
        <PopoverButton
            class="flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-indigo-600"
            :aria-label="t('projects.table_icon.change')"
        >
            <component :is="iconFor(icon)" class="h-6 w-6" aria-hidden="true" />
        </PopoverButton>

        <PopoverPanel
            v-slot="{ close }"
            class="absolute left-0 top-full z-50 mt-1 grid w-max grid-cols-5 gap-1 rounded-lg bg-white p-2 shadow-lg ring-1 ring-black/5"
        >
            <button
                v-for="(component, name, index) in tableIcons"
                :key="name"
                type="button"
                :aria-label="t('projects.table_icon.option', { number: index + 1 })"
                :aria-pressed="name === selected"
                :class="[
                    'flex items-center justify-center rounded p-1.5 hover:bg-indigo-50',
                    name === selected ? 'bg-indigo-100 text-indigo-600' : 'text-gray-500',
                ]"
                @click="choose(name, close)"
            >
                <component :is="component" class="h-5 w-5" aria-hidden="true" />
            </button>
        </PopoverPanel>
    </Popover>
</template>

<script setup>
import { computed } from "vue";
import { Popover, PopoverButton, PopoverPanel } from "@headlessui/vue";
import { t } from "@/i18n/index.js";
import { tableIcons } from "@/constants/tableIcons";

const props = defineProps({
    icon: {
        type: String,
        default: "",
    },
});

const emit = defineEmits(["select"]);

const selected = computed(() => (tableIcons[props.icon] ? props.icon : "TableCellsIcon"));

function iconFor(name) {
    return tableIcons[name] || tableIcons.TableCellsIcon;
}

function choose(name, close) {
    close();
    if (name !== selected.value) emit("select", name);
}
</script>
