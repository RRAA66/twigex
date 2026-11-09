// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { ref, toValue } from "vue";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";

const IMAGE_FIELDS = { cover: "cover_image", logo: "logo" };

// Each image is stored at once, apart from the form's other settings, which
// wait for Update. onChange gets the stored image unless another form was
// opened while it was sent.
export function useFormImages(form, onChange) {
    const alertStore = useAlertStore();

    const busy = ref("");

    async function change(kind, request) {
        const sent = toValue(form);

        busy.value = kind;
        try {
            const { data } = await request(sent);

            if (toValue(form).id !== sent.id) return;

            const field = IMAGE_FIELDS[kind];

            onChange({ [field]: data[field] || "" });
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
