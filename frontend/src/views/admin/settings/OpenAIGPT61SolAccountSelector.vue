<template>
  <div ref="containerRef" class="relative">
    <div v-if="selectedAccountIds.length > 0" class="mb-2 flex flex-wrap gap-2">
      <span
        v-for="accountId in selectedAccountIds"
        :key="accountId"
        class="inline-flex max-w-full items-center gap-1.5 rounded-md bg-gray-100 px-2.5 py-1.5 text-xs text-gray-700 dark:bg-dark-600 dark:text-gray-200"
      >
        <span class="max-w-64 truncate font-medium" :title="selectedAccountLabel(accountId)">
          {{ selectedAccountLabel(accountId) }}
        </span>
        <span class="shrink-0 text-gray-400">#{{ accountId }}</span>
        <span
          v-if="selectedAccounts[accountId]?.missing"
          class="shrink-0 text-gray-400"
        >
          {{ t("admin.settings.openaiGPT61SolLongContextBilling.accountMissing") }}
        </span>
        <button
          type="button"
          class="shrink-0 rounded text-gray-400 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:text-red-400"
          :disabled="disabled"
          :aria-label="t('admin.settings.openaiGPT61SolLongContextBilling.removeAccount')"
          :title="t('admin.settings.openaiGPT61SolLongContextBilling.removeAccount')"
          @click="removeAccount(accountId)"
        >
          <Icon name="x" size="xs" :stroke-width="2" />
        </button>
      </span>
    </div>

    <div class="relative">
      <Icon
        name="search"
        size="sm"
        class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
      />
      <input
        v-model="searchQuery"
        type="text"
        autocomplete="off"
        class="input input-sm w-full pl-9"
        :disabled="disabled"
        :placeholder="t('admin.settings.openaiGPT61SolLongContextBilling.accountSearchPlaceholder')"
        @input="debounceSearch"
        @focus="onFocus"
      />
    </div>

    <div
      v-if="showDropdown && searchQuery.trim()"
      class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-700"
    >
      <div v-if="searchLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
        {{ t("common.loading") }}
      </div>
      <div
        v-else-if="availableResults.length === 0"
        class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t("admin.settings.openaiGPT61SolLongContextBilling.accountSearchEmpty") }}
      </div>
      <template v-else>
        <button
          v-for="account in availableResults"
          :key="account.id"
          type="button"
          class="flex w-full items-center justify-between gap-3 px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-600"
          @click="selectAccount(account)"
        >
          <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">
            {{ account.name }}
          </span>
          <span class="shrink-0 text-xs text-gray-400">#{{ account.id }}</span>
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api";
import type { AccountListItem } from "@/types";
import Icon from "@/components/icons/Icon.vue";

interface SelectableAccount {
  id: number;
  name: string;
}

interface SelectedAccount extends SelectableAccount {
  missing: boolean;
}

const props = withDefaults(
  defineProps<{
    modelValue: number[];
    disabled?: boolean;
  }>(),
  { disabled: false },
);

const emit = defineEmits<{
  "update:modelValue": [value: number[]];
}>();

const { t } = useI18n();
const containerRef = ref<HTMLElement | null>(null);
const searchQuery = ref("");
const searchResults = ref<SelectableAccount[]>([]);
const searchLoading = ref(false);
const showDropdown = ref(false);
const selectedAccounts = ref<Record<number, SelectedAccount>>({});
let searchTimer: ReturnType<typeof setTimeout> | null = null;
let searchSequence = 0;

// 订阅账号（oauth / setup-token）本就豁免、影子账号的计费落在母账号，
// 列入名单会产生误导，故搜索结果中一律排除。
function isEligibleAccount(account: AccountListItem): boolean {
  if (account.type === "oauth" || account.type === "setup-token") {
    return false;
  }
  return !account.parent_account_id;
}

const selectedAccountIds = computed(() =>
  Array.from(
    new Set(
      (Array.isArray(props.modelValue) ? props.modelValue : []).filter(
        (id) => Number.isInteger(id) && id > 0,
      ),
    ),
  ),
);

const availableResults = computed(() => {
  const selected = new Set(selectedAccountIds.value);
  return searchResults.value.filter((account) => !selected.has(account.id));
});

function selectedAccountLabel(accountId: number): string {
  return (
    selectedAccounts.value[accountId]?.name ||
    t("admin.settings.openaiGPT61SolLongContextBilling.accountIdFallback", {
      id: accountId,
    })
  );
}

function clearPendingSearch(): void {
  if (searchTimer) {
    clearTimeout(searchTimer);
    searchTimer = null;
  }
  searchSequence += 1;
}

function onFocus(): void {
  if (!props.disabled) {
    showDropdown.value = true;
  }
}

function debounceSearch(): void {
  clearPendingSearch();
  if (props.disabled) {
    searchResults.value = [];
    searchLoading.value = false;
    showDropdown.value = false;
    return;
  }

  const query = searchQuery.value.trim();
  showDropdown.value = true;
  if (!query) {
    searchResults.value = [];
    searchLoading.value = false;
    return;
  }

  const sequence = searchSequence;
  searchTimer = setTimeout(async () => {
    searchLoading.value = true;
    try {
      const response = await adminAPI.accounts.list(1, 20, {
        platform: "openai",
        search: query,
      });
      if (sequence === searchSequence) {
        searchResults.value = response.items.filter(isEligibleAccount);
      }
    } catch {
      if (sequence === searchSequence) {
        searchResults.value = [];
      }
    } finally {
      if (sequence === searchSequence) {
        searchLoading.value = false;
      }
    }
  }, 300);
}

function selectAccount(account: SelectableAccount): void {
  selectedAccounts.value = {
    ...selectedAccounts.value,
    [account.id]: { id: account.id, name: account.name, missing: false },
  };
  emit("update:modelValue", [...selectedAccountIds.value, account.id]);
  clearPendingSearch();
  searchQuery.value = "";
  searchResults.value = [];
  searchLoading.value = false;
  showDropdown.value = false;
}

function removeAccount(accountId: number): void {
  if (props.disabled) return;
  emit(
    "update:modelValue",
    selectedAccountIds.value.filter((id) => id !== accountId),
  );
}

async function hydrateSelectedAccounts(accountIds: number[]): Promise<void> {
  const missing = accountIds.filter((id) => !selectedAccounts.value[id]);
  if (missing.length === 0) return;

  const accounts = await Promise.all(
    missing.map(async (id) => {
      try {
        const account = await adminAPI.accounts.getById(id);
        return {
          id: account.id,
          name: account.name,
          missing: false,
        } satisfies SelectedAccount;
      } catch {
        return {
          id,
          name: "",
          missing: true,
        } satisfies SelectedAccount;
      }
    }),
  );

  const next = { ...selectedAccounts.value };
  for (const account of accounts) {
    if (props.modelValue.includes(account.id)) {
      next[account.id] = account;
    }
  }
  selectedAccounts.value = next;
}

function handleDocumentClick(event: MouseEvent): void {
  const target = event.target as Node | null;
  if (target && !containerRef.value?.contains(target)) {
    showDropdown.value = false;
  }
}

watch(
  selectedAccountIds,
  (accountIds) => {
    void hydrateSelectedAccounts(accountIds);
  },
  { immediate: true },
);

onMounted(() => {
  document.addEventListener("click", handleDocumentClick);
});

onUnmounted(() => {
  clearPendingSearch();
  document.removeEventListener("click", handleDocumentClick);
});
</script>
