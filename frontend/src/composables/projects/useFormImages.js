// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { computed, onScopeDispose, ref, toValue } from "vue";
import { t } from "@/i18n/index.js";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { formImageUrl } from "@/utils/projects/forms";

const IMAGE_FIELDS = { cover: "cover_image", logo: "logo" };
// The server takes an image by its name's extension and these at most.
const IMAGE_EXTENSIONS = [".jpg", ".jpeg", ".png", ".gif", ".webp"];
const MAX_IMAGE_BYTES = 5 << 20;

// A file named as an image may not be one, which only loading it shows.
function loadsAsImage(url) {
    return new Promise((resolve) => {
        const image = new Image();

        image.onload = () => resolve(true);
        image.onerror = () => resolve(false);
        image.src = url;
    });
}

// A picked or removed image waits in the browser until the form is saved, as
// the form's other settings do, so the builder and its preview show it before
// anyone filling the form does.
export function useFormImages(form) {
    const alertStore = useAlertStore();

    // By kind: the picked file, or null for a removed image.
    const pending = ref({});
    const urls = {};
    // By kind: counts picks and removals, so a pick still loading gives way
    // to whatever came after it.
    const turns = {};

    function nextTurn(kind) {
        turns[kind] = (turns[kind] || 0) + 1;

        return turns[kind];
    }

    function forget(kind) {
        if (urls[kind]) URL.revokeObjectURL(urls[kind]);

        delete urls[kind];
    }

    function unpend(kind) {
        const rest = { ...pending.value };

        delete rest[kind];
        pending.value = rest;
    }

    // Refused here rather than on save, when the picker is long closed.
    function refusal(file) {
        const dot = file.name.lastIndexOf(".");
        const extension = dot < 0 ? "" : file.name.slice(dot).toLowerCase();

        if (!IMAGE_EXTENSIONS.includes(extension)) {
            return t.value("projects.form_view.error.image_type");
        }

        if (file.size > MAX_IMAGE_BYTES) return t.value("projects.form_view.error.image_too_large");

        return "";
    }

    async function pick(kind, file) {
        const reason = refusal(file);

        if (reason) {
            alertStore.showError(reason);

            return;
        }

        const turn = nextTurn(kind);
        const url = URL.createObjectURL(file);
        const loads = await loadsAsImage(url);

        if (turns[kind] !== turn || !loads) {
            URL.revokeObjectURL(url);
            if (turns[kind] === turn) {
                alertStore.showError(t.value("projects.form_view.error.image_unreadable"));
            }

            return;
        }

        forget(kind);
        urls[kind] = url;
        pending.value = { ...pending.value, [kind]: file };
    }

    function remove(kind) {
        nextTurn(kind);
        forget(kind);
        pending.value = { ...pending.value, [kind]: null };
    }

    function src(kind) {
        if (kind in pending.value) return urls[kind] || "";

        const saved = toValue(form);

        return formImageUrl(saved.workspace_id, saved[IMAGE_FIELDS[kind]]);
    }

    function discard() {
        Object.keys(IMAGE_FIELDS).forEach(nextTurn);
        Object.keys(urls).forEach(forget);
        pending.value = {};
    }

    // An image that fails stays waiting so the next save sends it again, as
    // does one picked anew while the one before it was being sent. A form
    // opened meanwhile gets nothing of the one that was saved.
    async function save(saved, onStored) {
        const { workspace_id: ws, table_id: table, id } = saved;

        for (const [kind, file] of Object.entries(pending.value)) {
            const { data } = file
                ? await formService.setFormImage(ws, table, id, kind, file)
                : await formService.removeFormImage(ws, table, id, kind);

            if (toValue(form).id !== id) return;

            if (pending.value[kind] === file) {
                forget(kind);
                unpend(kind);
            }

            const field = IMAGE_FIELDS[kind];

            onStored({ [field]: data[field] || "" });
        }
    }

    const hasPending = computed(() => Object.keys(pending.value).length > 0);

    onScopeDispose(discard);

    return { pick, remove, src, discard, save, hasPending };
}
