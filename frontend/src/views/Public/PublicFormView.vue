<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="h-screen overflow-y-auto bg-gray-100 px-4 py-12">
        <div class="mx-auto max-w-2xl">
            <FormFill :token="route.params.token" live />
        </div>
    </div>
</template>

<script setup>
import { onMounted } from "vue";
import { useRoute } from "vue-router";
import { setLocale } from "@/i18n/index.js";
import { languages } from "@/i18n/languages.js";
import userService from "@/services/userService";
import { useUserStore } from "@/store/user";
import FormFill from "@/components/Projects/Form/FormFill.vue";

const route = useRoute();
const userStore = useUserStore();

// This page is outside MainLayout, which is what sets the user's language.
async function applyLanguage() {
    try {
        const { data } = await userService.preferences();

        userStore.setPreferences(data);
    } catch {
        // Without the user's choice the browser's language is the best guess.
    }

    const browser = (navigator.language || "").split("-")[0];
    const locale = userStore.getLanguage || (languages[browser] ? browser : "");

    if (locale) await setLocale(locale);
}

onMounted(applyLanguage);
</script>
