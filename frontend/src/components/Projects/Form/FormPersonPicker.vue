<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <FormLinkedPicker
        :model-value="picked"
        :loader="loadMembers"
        @update:model-value="emit('update:modelValue', $event?.id || '')"
    >
        <template #leading="{ item }">
            <span class="ml-0.5 size-5 shrink-0"><UserAvatar :user-id="item.id" /></span>
        </template>
    </FormLinkedPicker>
</template>

<script setup>
import { computed } from "vue";
import { useUserState } from "@/composables/useUser";
import workspaceService from "@/services/workspaceService";
import { useUserStore } from "@/store/user";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import UserAvatar from "@/components/UserAvatar.vue";

const props = defineProps({
    modelValue: { type: String, default: "" },
    workspaceId: { type: String, required: true },
});

const emit = defineEmits(["update:modelValue"]);

const userStore = useUserStore();

const { user } = useUserState(() => props.modelValue || null);

const picked = computed(() =>
    props.modelValue ? { id: props.modelValue, name: memberName(user.value) } : null,
);

function memberName(member) {
    if (!member) return "";

    return [member.name, member.lastname].filter(Boolean).join(" ") || member.username || "";
}

// The members are searched on the server, as a workspace can have thousands.
// One past the page tells whether another follows.
async function loadMembers({ page, limit, search }) {
    const { data } = await workspaceService.searchWorkspaceMembers(
        props.workspaceId,
        search || "",
        limit + 1,
        (page - 1) * limit,
    );
    const users = data || [];

    userStore.addUsers(users);

    return {
        items: users.slice(0, limit).map((u) => ({ id: u.id, name: memberName(u) })),
        has_more: users.length > limit,
    };
}
</script>
