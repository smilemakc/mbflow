<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter } from "vue-router";
import { Icon } from "@iconify/vue";
import AppShell from "@/components/layout/AppShell.vue";
import Button from "@/components/ui/Button.vue";
import Badge from "@/components/ui/Badge.vue";
import { workflowTemplates } from "@/data/templates";
import type {
  WorkflowTemplate,
  TemplateCategory,
} from "@/types/workflowTemplate";

const router = useRouter();

const searchQuery = ref("");
const selectedCategory = ref<TemplateCategory | "all">("all");
const selectedDifficulty = ref<
  "all" | "beginner" | "intermediate" | "advanced"
>("all");

// Filter templates
const filteredTemplates = computed(() => {
  let templates = workflowTemplates;

  // Filter by search query
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    templates = templates.filter(
      (t) =>
        t.name.toLowerCase().includes(query) ||
        t.description.toLowerCase().includes(query) ||
        t.tags.some((tag) => tag.toLowerCase().includes(query)),
    );
  }

  // Filter by category
  if (selectedCategory.value !== "all") {
    templates = templates.filter((t) => t.category === selectedCategory.value);
  }

  // Filter by difficulty
  if (selectedDifficulty.value !== "all") {
    templates = templates.filter(
      (t) => t.difficulty === selectedDifficulty.value,
    );
  }

  return templates;
});

function getCategoryLabel(category: TemplateCategory): string {
  const labels: Record<TemplateCategory, string> = {
    "data-processing": "Data Processing",
    "api-integration": "API Integration",
    "ai-automation": "AI Automation",
    notification: "Notification",
    monitoring: "Monitoring",
    etl: "ETL",
    other: "Other",
  };
  return labels[category];
}

function getDifficultyColor(difficulty: string): string {
  const colors: Record<string, string> = {
    beginner: "bg-green-100 text-green-700",
    intermediate: "bg-yellow-100 text-yellow-700",
    advanced: "bg-red-100 text-red-700",
  };
  return colors[difficulty] || "bg-gray-100 text-gray-700";
}

function handleUseTemplate(template: WorkflowTemplate) {
  // Navigate to workflow creation with template ID
  router.push({
    path: "/workflows/new",
    query: { template: template.id },
  });
}

function handleViewTemplate(template: WorkflowTemplate) {
  // For now, just use the template
  // In the future, could show a preview modal
  handleUseTemplate(template);
}
</script>

<template>
  <AppShell>
    <div class="mx-auto max-w-7xl">
      <!-- Header -->
      <div class="mb-6">
        <h1 class="text-3xl font-bold text-gray-900">Workflow Templates</h1>
        <p class="mt-2 text-gray-600">
          Start with a pre-built template and customize it for your needs
        </p>
      </div>

      <!-- Filters -->
      <div class="card mb-6">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
          <!-- Search -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Search
            </label>
            <div class="relative">
              <Icon
                icon="heroicons:magnifying-glass"
                class="absolute left-3 top-1/2 size-5 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="searchQuery"
                type="text"
                placeholder="Search templates..."
                class="w-full rounded-md border border-gray-300 py-2 pl-10 pr-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <!-- Category -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Category
            </label>
            <select
              v-model="selectedCategory"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Categories</option>
              <option value="data-processing">Data Processing</option>
              <option value="api-integration">API Integration</option>
              <option value="ai-automation">AI Automation</option>
              <option value="notification">Notification</option>
              <option value="monitoring">Monitoring</option>
              <option value="etl">ETL</option>
              <option value="other">Other</option>
            </select>
          </div>

          <!-- Difficulty -->
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              Difficulty
            </label>
            <select
              v-model="selectedDifficulty"
              class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Levels</option>
              <option value="beginner">Beginner</option>
              <option value="intermediate">Intermediate</option>
              <option value="advanced">Advanced</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Results count -->
      <div class="mb-4 text-sm text-gray-600">
        {{ filteredTemplates.length }} template{{
          filteredTemplates.length !== 1 ? "s" : ""
        }}
        found
      </div>

      <!-- Templates grid -->
      <div
        v-if="filteredTemplates.length > 0"
        class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3"
      >
        <div
          v-for="template in filteredTemplates"
          :key="template.id"
          class="card group cursor-pointer transition-all hover:shadow-lg"
          @click="handleViewTemplate(template)"
        >
          <!-- Header -->
          <div class="mb-4 flex items-start justify-between">
            <div class="flex items-center gap-3">
              <div
                class="flex size-12 items-center justify-center rounded-lg bg-blue-100"
              >
                <Icon
                  :icon="template.icon || 'heroicons:document-duplicate'"
                  class="size-6 text-blue-600"
                />
              </div>
              <div>
                <h3
                  class="font-semibold text-gray-900 group-hover:text-blue-600"
                >
                  {{ template.name }}
                </h3>
                <p class="text-xs text-gray-500">
                  {{ template.nodes.length }} nodes
                </p>
              </div>
            </div>
          </div>

          <!-- Description -->
          <p class="mb-4 line-clamp-2 text-sm text-gray-600">
            {{ template.description }}
          </p>

          <!-- Metadata -->
          <div class="mb-4 flex flex-wrap gap-2">
            <Badge variant="gray">
              {{ getCategoryLabel(template.category) }}
            </Badge>
            <span
              :class="[
                getDifficultyColor(template.difficulty),
                'rounded px-2 py-0.5 text-xs font-medium',
              ]"
            >
              {{ template.difficulty }}
            </span>
          </div>

          <!-- Tags -->
          <div class="mb-4 flex flex-wrap gap-1">
            <span
              v-for="tag in template.tags.slice(0, 3)"
              :key="tag"
              class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
            >
              #{{ tag }}
            </span>
            <span
              v-if="template.tags.length > 3"
              class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
            >
              +{{ template.tags.length - 3 }}
            </span>
          </div>

          <!-- Actions -->
          <div class="flex gap-2 border-t border-gray-200 pt-4">
            <Button
              variant="primary"
              size="sm"
              class="flex-1"
              @click.stop="handleUseTemplate(template)"
            >
              <Icon icon="heroicons:plus" class="mr-1 size-4" />
              Use Template
            </Button>
            <Button
              variant="secondary"
              size="sm"
              @click.stop="handleViewTemplate(template)"
            >
              <Icon icon="heroicons:eye" class="size-4" />
            </Button>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else class="card py-12 text-center">
        <Icon
          icon="heroicons:document-magnifying-glass"
          class="mx-auto size-12 text-gray-400"
        />
        <h3 class="mt-2 text-sm font-medium text-gray-900">
          No templates found
        </h3>
        <p class="mt-1 text-sm text-gray-500">
          Try adjusting your filters or search query
        </p>
        <div class="mt-6">
          <Button
            variant="secondary"
            @click="
              () => {
                searchQuery = '';
                selectedCategory = 'all';
                selectedDifficulty = 'all';
              }
            "
          >
            Clear Filters
          </Button>
        </div>
      </div>
    </div>
  </AppShell>
</template>
