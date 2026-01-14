"use client";

import { StatCard } from "@/components/ui-custom/StatCard";
import { Link2, MousePointerClick, Users } from "lucide-react";
import { DashboardResponse } from "@/lib/api";

interface StatsCardsProps {
  stats: DashboardResponse | null;
  loading?: boolean;
}

export function StatsCards({ stats, loading = false }: StatsCardsProps) {
  return (
    <div className="grid gap-4 md:grid-cols-3">
      <StatCard
        title="Total URLs"
        value={stats?.total_urls ?? 0}
        icon={Link2}
        loading={loading}
        iconClassName="bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"
      />
      <StatCard
        title="Total Clicks"
        value={stats?.total_clicks ?? 0}
        icon={MousePointerClick}
        loading={loading}
        iconClassName="bg-green-100 text-green-600 dark:bg-green-900/20 dark:text-green-400"
      />
      <StatCard
        title="Unique Visitors"
        value={stats?.unique_visitors ?? 0}
        icon={Users}
        loading={loading}
        iconClassName="bg-purple-100 text-purple-600 dark:bg-purple-900/20 dark:text-purple-400"
      />
    </div>
  );
}
