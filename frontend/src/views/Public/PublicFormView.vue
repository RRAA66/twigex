<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="h-screen overflow-y-auto bg-gray-100 px-4 py-12">
        <div class="mx-auto max-w-2xl">
            <FormFill :token="route.params.token" live follow-redirect />
        </div>
    </div>
</template>

<script setup>
import { useRoute } from "vue-router";
import FormFill from "@/components/Projects/Form/FormFill.vue";

const route = useRoute();
</script>
