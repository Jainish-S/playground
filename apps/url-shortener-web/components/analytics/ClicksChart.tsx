"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ClicksTimeSeriesResponse } from "@/lib/api";
import { format } from "date-fns";

interface ClicksChartProps {
  data: ClicksTimeSeriesResponse | null;
  loading?: boolean;
}

export function ClicksChart({ data, loading = false }: ClicksChartProps) {
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
          <CardTitle>Clicks Over Time</CardTitle>
          <CardDescription>No data available</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-[300px] flex items-center justify-center text-muted-foreground">
            No click data yet
          </div>
        </CardContent>
      </Card>
    );
  }

  // Format the data for Recharts
  const chartData = data.data.map((item) => ({
    date: format(new Date(item.bucket), "MMM dd"),
    clicks: item.clicks,
    unique: item.unique,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle>Clicks Over Time</CardTitle>
        <CardDescription>Total and unique clicks over the past {data.days} days</CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis
              dataKey="date"
              tick={{ fontSize: 12 }}
              className="text-muted-foreground"
            />
            <YAxis
              tick={{ fontSize: 12 }}
              className="text-muted-foreground"
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "hsl(var(--background))",
                border: "1px solid hsl(var(--border))",
                borderRadius: "8px",
              }}
              labelStyle={{ color: "hsl(var(--foreground))" }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="clicks"
              stroke="hsl(var(--primary))"
              strokeWidth={2}
              dot={{ fill: "hsl(var(--primary))", r: 4 }}
              activeDot={{ r: 6 }}
              name="Total Clicks"
            />
            <Line
              type="monotone"
              dataKey="unique"
              stroke="hsl(142.1 76.2% 36.3%)"
              strokeWidth={2}
              dot={{ fill: "hsl(142.1 76.2% 36.3%)", r: 4 }}
              activeDot={{ r: 6 }}
              name="Unique Visitors"
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
