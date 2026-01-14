"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ExternalLink, TrendingUp, Users, Smartphone } from "lucide-react";
import { CopyButton } from "@/components/ui-custom/CopyButton";
import { StatCard } from "@/components/ui-custom/StatCard";
import { ClicksChart } from "@/components/analytics/ClicksChart";
import { GeoBreakdown } from "@/components/analytics/GeoBreakdown";
import { DevicePieChart } from "@/components/analytics/DevicePieChart";
import { QRCodeDisplay } from "@/components/analytics/QRCodeDisplay";
import {
  URLResponse,
  AnalyticsResponse,
  ClicksTimeSeriesResponse,
  GeoBreakdownResponse,
  DeviceBreakdownResponse,
} from "@/lib/api";
import { formatDistanceToNow } from "date-fns";
import {
  getURLAction,
  getAnalyticsAction,
  getClicksTimeSeriesAction,
  getGeoBreakdownAction,
  getDeviceBreakdownAction,
} from "@/app/actions";

interface URLDetailsSheetProps {
  urlId: string | null;
  onClose?: () => void;
}

export function URLDetailsSheet({ urlId, onClose }: URLDetailsSheetProps) {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [url, setUrl] = useState<URLResponse | null>(null);
  const [analytics, setAnalytics] = useState<AnalyticsResponse | null>(null);
  const [timeSeries, setTimeSeries] = useState<ClicksTimeSeriesResponse | null>(null);
  const [geo, setGeo] = useState<GeoBreakdownResponse | null>(null);
  const [devices, setDevices] = useState<DeviceBreakdownResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const open = !!urlId;

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      // Remove the 'view' query param to close the sheet
      const params = new URLSearchParams(searchParams.toString());
      params.delete("view");
      router.push(`?${params.toString()}`, { scroll: false });
      if (onClose) onClose();
    }
  };

  useEffect(() => {
    if (!urlId) {
      setUrl(null);
      setAnalytics(null);
      setTimeSeries(null);
      setGeo(null);
      setDevices(null);
      setError(null);
      return;
    }

    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);

        // Fetch all analytics data in parallel
        const [urlResult, analyticsResult, timeSeriesResult, geoResult, devicesResult] = await Promise.all([
          getURLAction(urlId),
          getAnalyticsAction(urlId),
          getClicksTimeSeriesAction(urlId, 7),
          getGeoBreakdownAction(urlId),
          getDeviceBreakdownAction(urlId),
        ]);

        if (urlResult.success && urlResult.data) {
          setUrl(urlResult.data);
        }
        if (analyticsResult.success && analyticsResult.data) {
          setAnalytics(analyticsResult.data);
        }
        if (timeSeriesResult.success && timeSeriesResult.data) {
          setTimeSeries(timeSeriesResult.data);
        }
        if (geoResult.success && geoResult.data) {
          setGeo(geoResult.data);
        }
        if (devicesResult.success && devicesResult.data) {
          setDevices(devicesResult.data);
        }

        // Check if any request failed
        if (!urlResult.success) {
          throw new Error(urlResult.error || "Failed to load URL data");
        }
      } catch (err: any) {
        console.error("Failed to fetch URL analytics:", err);
        setError(err.message || "Failed to load analytics data");
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [urlId]);

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className="w-full sm:max-w-4xl overflow-y-auto">
        {(!url && !error) ? (
          <div className="space-y-4">
            <SheetHeader>
              <SheetTitle>Loading Analytics</SheetTitle>
              <SheetDescription asChild>
                <div>
                  <Skeleton className="h-4 w-96" />
                </div>
              </SheetDescription>
            </SheetHeader>
            <div className="grid gap-4 md:grid-cols-3">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
            <Skeleton className="h-[300px] w-full" />
          </div>
        ) : error ? (
          <div className="space-y-4">
            <SheetHeader>
              <SheetTitle>Error Loading Analytics</SheetTitle>
              <SheetDescription className="text-destructive">{error}</SheetDescription>
            </SheetHeader>
          </div>
        ) : url ? (
          <div className="space-y-6">
            {/* Header */}
            <SheetHeader>
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1 min-w-0">
                  <SheetTitle className="text-2xl font-bold break-all">
                    {url.short_url}
                  </SheetTitle>
                  <SheetDescription className="mt-2 break-all">
                    {url.destination_url}
                  </SheetDescription>
                </div>
                <Badge variant={url.is_active ? "default" : "secondary"}>
                  {url.is_active ? "Active" : "Inactive"}
                </Badge>
              </div>

              {/* Quick Actions */}
              <div className="flex items-center gap-2 pt-2">
                <CopyButton
                  value={url.short_url}
                  variant="outline"
                  size="sm"
                  showLabel
                  successMessage="Short URL copied!"
                />
                <Button
                  variant="outline"
                  size="sm"
                  asChild
                >
                  <a href={url.destination_url} target="_blank" rel="noopener noreferrer">
                    <ExternalLink className="h-4 w-4 mr-2" />
                    Visit
                  </a>
                </Button>
              </div>

              {/* Metadata */}
              <div className="text-xs text-muted-foreground pt-2 space-y-1">
                <p>Created {formatDistanceToNow(new Date(url.created_at))} ago</p>
                {url.notes && <p className="italic">Note: {url.notes}</p>}
              </div>
            </SheetHeader>

            {/* Key Metrics */}
            <div className="grid gap-4 md:grid-cols-3">
              <StatCard
                title="Total Clicks"
                value={analytics?.total_clicks ?? 0}
                icon={TrendingUp}
                loading={loading}
                iconClassName="bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"
              />
              <StatCard
                title="Unique Visitors"
                value={analytics?.unique_visitors ?? 0}
                icon={Users}
                loading={loading}
                iconClassName="bg-green-100 text-green-600 dark:bg-green-900/20 dark:text-green-400"
              />
              <StatCard
                title="Mobile Clicks"
                value={analytics?.mobile_clicks ?? 0}
                icon={Smartphone}
                loading={loading}
                iconClassName="bg-purple-100 text-purple-600 dark:bg-purple-900/20 dark:text-purple-400"
              />
            </div>

            {/* Analytics Tabs */}
            <Tabs defaultValue="overview" className="w-full">
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="geographic">Geographic</TabsTrigger>
                <TabsTrigger value="devices">Devices</TabsTrigger>
                <TabsTrigger value="qrcode">QR Code</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-4 mt-4">
                <ClicksChart data={timeSeries} loading={loading} />
              </TabsContent>

              <TabsContent value="geographic" className="space-y-4 mt-4">
                <GeoBreakdown data={geo} loading={loading} />
              </TabsContent>

              <TabsContent value="devices" className="space-y-4 mt-4">
                <DevicePieChart data={devices} loading={loading} />
              </TabsContent>

              <TabsContent value="qrcode" className="space-y-4 mt-4">
                <QRCodeDisplay
                  url={url.short_url}
                  fileName={`qr-${url.short_code}`}
                  size={220}
                />
              </TabsContent>
            </Tabs>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
