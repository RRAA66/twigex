<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <Combobox
        v-slot="{ open }"
        as="div"
        by="id"
        :model-value="multiple ? modelValue || [] : modelValue"
        :multiple="multiple"
        :nullable="!multiple"
        @update:model-value="emit('update:modelValue', $event)"
    >
        <div
            ref="anchor"
            @focusin="loadFirstPage"
            class="relative flex w-full flex-wrap items-center gap-1 rounded-md bg-white py-1 pl-2 shadow-sm ring-1 ring-inset focus-within:ring-2 focus-within:ring-indigo-600"
            :class="[error ? 'ring-red-600' : 'ring-gray-300', picked ? 'pr-16' : 'pr-10']"
        >
            <span
                v-for="item in chips"
                :key="item.id"
                class="inline-flex items-center gap-x-1 rounded-md bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700"
            >
                <span
                    v-if="item.color"
                    class="h-2 w-2 flex-none rounded-full"
                    :class="getStatusStyle(item).class"
                    :style="getStatusStyle(item).style"
                    aria-hidden="true"
                />
                {{ item.name }}
                <button
                    type="button"
                    class="text-indigo-400 hover:text-indigo-600"
                    @click="remove(item)"
                >
                    <span class="sr-only">{{ t("common.button.remove") }}</span>
                    <XMarkIcon class="h-3.5 w-3.5" aria-hidden="true" />
                </button>
            </span>
            <slot v-if="picked" name="leading" :item="picked">
                <span
                    v-if="picked.color"
                    class="ml-1 h-2 w-2 flex-none rounded-full"
                    :class="getStatusStyle(picked).class"
                    :style="getStatusStyle(picked).style"
                    aria-hidden="true"
                />
            </slot>
            <ComboboxInput
                :id="id"
                v-bind="$attrs"
                class="min-w-[6rem] flex-1 border-0 bg-transparent px-1 py-0.5 text-sm text-gray-900 placeholder:text-gray-400 focus:ring-0"
                :display-value="(item) => (multiple ? '' : item?.name || '')"
                :placeholder="chips.length ? '' : placeholder"
                @change="search = $event.target.value"
                @click="open || openList()"
            />
            <button
                v-if="picked"
                type="button"
                class="absolute inset-y-0 right-8 flex items-center px-1 text-gray-400 hover:text-gray-600"
                @click="emit('update:modelValue', null)"
            >
                <span class="sr-only">{{ t("common.button.clear") }}</span>
                <XMarkIcon class="h-4 w-4" aria-hidden="true" />
            </button>
            <ComboboxButton
                ref="button"
                class="absolute inset-y-0 right-0 flex items-center px-2"
                @click="loadFirstPage"
            >
                <ChevronUpDownIcon class="h-5 w-5 text-gray-400" aria-hidden="true" />
            </ComboboxButton>
        </div>

        <Portal>
            <ComboboxOptions
                ref="floating"
                :style="floatingStyles"
                class="z-[60] max-h-60 overflow-auto rounded-md bg-white py-1 text-sm shadow-lg ring-1 ring-black/5 focus:outline-none"
            >
                <li v-if="loading" class="flex justify-center px-3 py-2">
                    <BaseSpinner
                        size="sm"
                        class="text-indigo-600"
                        :label="t('common.label.loading')"
                    />
                </li>
                <li v-else-if="loadFailed" class="px-3 py-2 text-red-600">
                    {{ t("projects.form_view.error.linked_options_load_failed") }}
                </li>
                <li v-else-if="items.length === 0" class="px-3 py-2 text-gray-500">
                    {{ t("projects.form_view.no_options") }}
                </li>
                <ComboboxOption
                    v-for="item in items"
                    v-slot="{ active, selected }"
                    :key="item.id"
                    :value="item"
                    as="template"
                >
                    <li
                        :class="[
                            active ? 'bg-indigo-600 text-white' : 'text-gray-900',
                            'relative cursor-default select-none py-2 pl-3 pr-9',
                        ]"
                    >
                        <span class="flex items-center gap-x-2">
                            <slot name="leading" :item="item">
                                <span
                                    v-if="item.color"
                                    class="h-2 w-2 flex-none rounded-full"
                                    :class="getStatusStyle(item).class"
                                    :style="getStatusStyle(item).style"
                                    aria-hidden="true"
                                />
                            </slot>
                            <span :class="[selected ? 'font-semibold' : '', 'truncate']">
                                {{ item.name }}
                            </span>
                        </span>
                        <CheckIcon
                            v-if="selected"
                            :class="[
                                active ? 'text-white' : 'text-indigo-600',
                                'absolute inset-y-0 right-3 my-auto h-5 w-5',
                            ]"
                            aria-hidden="true"
                        />
                    </li>
                </ComboboxOption>
                <li v-if="!loading && hasMore" class="px-3 py-1">
                    <BaseButton
                        type="button"
                        size="small"
                        variant="secondary"
                        class="w-full"
                        :is-loading="loadingMore"
                        @click="fetchPage(page + 1)"
                    >
                        {{ t("projects.form_view.load_more") }}
                    </BaseButton>
                </li>
            </ComboboxOptions>
        </Portal>
    </Combobox>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from "vue";
import {
    Combobox,
    ComboboxButton,
    ComboboxInput,
    ComboboxOption,
    ComboboxOptions,
    Portal,
} from "@headlessui/vue";
import { CheckIcon, ChevronUpDownIcon, XMarkIcon } from "@heroicons/vue/20/solid";
import { t } from "@/i18n/index.js";
import { useAnchoredPopup } from "@/composables/useAnchoredPopup";
import formService from "@/services/formService";
import { getStatusStyle } from "@/utils/projects/cells";
import BaseButton from "@/components/BaseButton.vue";
import BaseSpinner from "@/components/BaseSpinner.vue";

defineOptions({ inheritAttrs: false });

const PAGE_SIZE = 50;
const SEARCH_DELAY_MS = 250;

const props = defineProps({
    modelValue: { type: [Object, Array], default: null },
    token: { type: String, default: "" },
    tableId: { type: String, default: "" },
    // Lists the options in place of the form link, given { page, limit,
    // search } and answering { items, has_more }.
    loader: { type: Function, default: null },
    multiple: { type: Boolean, default: false },
    placeholder: { type: String, default: "" },
    id: { type: String, default: undefined },
    error: { type: Boolean, default: false },
});

const emit = defineEmits(["update:modelValue"]);

const anchor = ref(null);
const button = ref(null);
const { floating, floatingStyles } = useAnchoredPopup({ matchWidth: true, anchor });

const search = ref("");
const items = ref([]);
const hasMore = ref(false);
const page = ref(0);
const loading = ref(false);
const loadingMore = ref(false);
const loadFailed = ref(false);
let request = 0;
let searchTimer = null;

const chips = computed(() => (props.multiple ? props.modelValue || [] : []));
const picked = computed(() => (props.multiple ? null : props.modelValue));

async function requestPage(next) {
    const params = { page: next, limit: PAGE_SIZE, search: search.value || undefined };

    if (props.loader) return props.loader(params);

    const { data } = await formService.getPublicLinkedItems(props.token, props.tableId, params);

    return data;
}

async function fetchPage(next) {
    const current = ++request;

    loadFailed.value = false;
    if (next === 1) loading.value = true;
    else loadingMore.value = true;

    try {
        const data = await requestPage(next);

        // A slower answer to an older search must not replace a newer one.
        if (current !== request) return;

        items.value = next === 1 ? data.items : [...items.value, ...data.items];
        hasMore.value = data.has_more;
        page.value = next;
    } catch {
        if (current === request) loadFailed.value = true;
    } finally {
        if (current === request) {
            loading.value = false;
            loadingMore.value = false;
        }
    }
}

// This Headless UI version ignores Combobox's `immediate`, so a click in the
// field opens the list through its button.
function openList() {
    button.value?.$el.click();
}

function loadFirstPage() {
    if (page.value === 0 && !loading.value) fetchPage(1);
}

function remove(item) {
    emit(
        "update:modelValue",
        chips.value.filter((i) => i.id !== item.id),
    );
}

watch(search, () => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => fetchPage(1), SEARCH_DELAY_MS);
});

onBeforeUnmount(() => clearTimeout(searchTimer));
</script>
