"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { GeoBreakdownResponse } from "@/lib/api";
import { Globe } from "lucide-react";

interface GeoBreakdownProps {
  data: GeoBreakdownResponse | null;
  loading?: boolean;
}

// Map country codes to flag emojis (partial list)
const countryFlags: { [key: string]: string } = {
  US: "🇺🇸", GB: "🇬🇧", CA: "🇨🇦", AU: "🇦🇺", DE: "🇩🇪",
  FR: "🇫🇷", JP: "🇯🇵", CN: "🇨🇳", IN: "🇮🇳", BR: "🇧🇷",
  IT: "🇮🇹", ES: "🇪🇸", MX: "🇲🇽", KR: "🇰🇷", NL: "🇳🇱",
};

export function GeoBreakdown({ data, loading = false }: GeoBreakdownProps) {
  if (loading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-60 mt-2" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[300px] w-full" />
        </CardContent>
      </Card>
    );
  }

  if (!data || !data.data || data.data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Geographic Breakdown
          </CardTitle>
          <CardDescription>No geographic data available</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-[300px] flex items-center justify-center text-muted-foreground">
            No geographic data yet
          </div>
        </CardContent>
      </Card>
    );
  }

  // Format data and add flags
  const chartData = data.data.map((item) => ({
    country: item.country,
    displayName: `${countryFlags[item.country] || "🌐"} ${item.country}`,
    clicks: item.clicks,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Globe className="h-5 w-5" />
          Geographic Breakdown
        </CardTitle>
        <CardDescription>Clicks by country</CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart
            data={chartData}
            layout="vertical"
            margin={{ top: 5, right: 10, left: 10, bottom: 5 }}
          >
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" horizontal={false} />
            <XAxis
              type="number"
              tick={{ fontSize: 12 }}
              className="text-muted-foreground"
            />
            <YAxis
              type="category"
              dataKey="displayName"
              tick={{ fontSize: 12 }}
              className="text-muted-foreground"
              width={80}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "hsl(var(--background))",
                border: "1px solid hsl(var(--border))",
                borderRadius: "8px",
              }}
              labelStyle={{ color: "hsl(var(--foreground))" }}
            />
            <Bar
              dataKey="clicks"
              fill="hsl(var(--primary))"
              radius={[0, 4, 4, 0]}
            />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
