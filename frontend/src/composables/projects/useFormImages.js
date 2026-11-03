// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { ref, toValue } from "vue";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";

const IMAGE_FIELDS = { cover: "cover_image", logo: "logo" };

// Uploads and removes a saved form's cover and logo. Each is stored at once,
// apart from the form's other settings, which wait for Update.
export function useFormImages(form) {
    const alertStore = useAlertStore();

    const busy = ref("");

    async function change(kind, request) {
        busy.value = kind;
        try {
            const { data } = await request(toValue(form));

            toValue(form)[IMAGE_FIELDS[kind]] = data[IMAGE_FIELDS[kind]] || "";
        } catch (error) {
            alertStore.showError(extractErrorMessage(error));
        } finally {
            busy.value = "";
        }
    }

    function setImage(kind, file) {
        return change(kind, (f) =>
            formService.setFormImage(f.workspace_id, f.table_id, f.id, kind, file),
        );
    }

    function removeImage(kind) {
        return change(kind, (f) =>
            formService.removeFormImage(f.workspace_id, f.table_id, f.id, kind),
        );
    }

    return { busy, setImage, removeImage };
}
