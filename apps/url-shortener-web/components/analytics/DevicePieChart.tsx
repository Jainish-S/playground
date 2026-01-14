"use client";

import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from "recharts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { DeviceBreakdownResponse } from "@/lib/api";
import { Smartphone } from "lucide-react";

interface DevicePieChartProps {
  data: DeviceBreakdownResponse | null;
  loading?: boolean;
}

// Color scheme for device types
const DEVICE_COLORS: { [key: string]: string } = {
  Mobile: "hsl(var(--primary))",
  Desktop: "hsl(142.1 76.2% 36.3%)",
  Tablet: "hsl(24.6 95% 53.1%)",
  Unknown: "hsl(var(--muted))",
};

export function DevicePieChart({ data, loading = false }: DevicePieChartProps) {
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

  if (!data || !data.devices || data.devices.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Smartphone className="h-5 w-5" />
            Device Distribution
          </CardTitle>
          <CardDescription>No device data available</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-[300px] flex items-center justify-center text-muted-foreground">
            No device data yet
          </div>
        </CardContent>
      </Card>
    );
  }

  // Format data for pie chart
  const chartData = data.devices.map((item) => ({
    name: item.device_type,
    value: item.clicks,
  }));

  // Calculate total clicks for percentages
  const totalClicks = chartData.reduce((sum, item) => sum + item.value, 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Smartphone className="h-5 w-5" />
          Device Distribution
        </CardTitle>
        <CardDescription>Clicks by device type</CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <PieChart>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              labelLine={false}
              label={({ name, value }) => {
                const percentage = ((value / totalClicks) * 100).toFixed(1);
                return `${name}: ${percentage}%`;
              }}
              outerRadius={90}
              fill="#8884d8"
              dataKey="value"
            >
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={DEVICE_COLORS[entry.name] || DEVICE_COLORS.Unknown} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: "hsl(var(--background))",
                border: "1px solid hsl(var(--border))",
                borderRadius: "8px",
              }}
              formatter={(value: number) => [`${value} clicks`, ""]}
            />
            <Legend
              verticalAlign="bottom"
              height={36}
              formatter={(value) => {
                const item = chartData.find((d) => d.name === value);
                if (!item) return value;
                const percentage = ((item.value / totalClicks) * 100).toFixed(1);
                return `${value} (${percentage}%)`;
              }}
            />
          </PieChart>
        </ResponsiveContainer>

        {/* Browser breakdown table */}
        {data.browsers && data.browsers.length > 0 && (
          <div className="mt-6">
            <h4 className="text-sm font-semibold mb-3">Browser Breakdown</h4>
            <div className="space-y-2">
              {data.browsers.slice(0, 5).map((browser, index) => {
                const percentage = ((browser.clicks / totalClicks) * 100).toFixed(1);
                return (
                  <div key={index} className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">{browser.browser}</span>
                    <div className="flex items-center gap-2">
                      <div className="w-24 h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary"
                          style={{ width: `${percentage}%` }}
                        />
                      </div>
                      <span className="font-medium w-12 text-right">{browser.clicks}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
