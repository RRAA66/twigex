// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { onScopeDispose, ref, toValue } from "vue";
import { t } from "@/i18n/index.js";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { formImageUrl } from "@/utils/projects/forms";

const IMAGE_FIELDS = { cover: "cover_image", logo: "logo" };
const IMAGE_TYPES = ["image/jpeg", "image/png", "image/gif", "image/webp"];
const MAX_IMAGE_BYTES = 5 << 20;

// A picked or removed image waits in the browser until the form is saved, as
// the form's other settings do, so the builder and its preview show it before
// anyone filling the form does.
export function useFormImages(form) {
    const alertStore = useAlertStore();

    // By kind: the picked file, or null for a removed image.
    const pending = ref({});
    const urls = {};

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
        if (!IMAGE_TYPES.includes(file.type)) return t.value("projects.form_view.error.image_type");

        if (file.size > MAX_IMAGE_BYTES) return t.value("projects.form_view.error.image_too_large");

        return "";
    }

    function pick(kind, file) {
        const reason = refusal(file);

        if (reason) {
            alertStore.showError(reason);

            return;
        }

        forget(kind);
        urls[kind] = URL.createObjectURL(file);
        pending.value = { ...pending.value, [kind]: file };
    }

    function remove(kind) {
        forget(kind);
        pending.value = { ...pending.value, [kind]: null };
    }

    function src(kind) {
        if (kind in pending.value) return urls[kind] || "";

        const saved = toValue(form);

        return formImageUrl(saved.workspace_id, saved[IMAGE_FIELDS[kind]]);
    }

    function discard() {
        Object.keys(urls).forEach(forget);
        pending.value = {};
    }

    // onStored gets each image as it is stored. One that fails stays waiting,
    // so the next save sends it again.
    async function save(saved, onStored) {
        for (const [kind, file] of Object.entries(pending.value)) {
            const { workspace_id: ws, table_id: table, id } = saved;
            const { data } = file
                ? await formService.setFormImage(ws, table, id, kind, file)
                : await formService.removeFormImage(ws, table, id, kind);
            const field = IMAGE_FIELDS[kind];

            forget(kind);
            unpend(kind);
            onStored({ [field]: data[field] || "" });
        }
    }

    onScopeDispose(discard);

    return { pending, pick, remove, src, discard, save };
}
