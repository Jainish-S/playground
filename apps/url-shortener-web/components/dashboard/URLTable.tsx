"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { CopyButton } from "@/components/ui-custom/CopyButton";
import { EmptyState } from "@/components/ui-custom/EmptyState";
import { URLActions } from "@/components/dashboard/URLActions";
import { URLResponse } from "@/lib/api";
import { formatDistance } from "date-fns";
import { ExternalLink, Link2, Filter, X, ChevronLeft, ChevronRight, ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Calendar } from "@/components/ui/calendar";
import { format } from "date-fns";
import { URLFilters } from "@/components/dashboard-content";

interface URLTableProps {
  urls: URLResponse[];
  total: number;
  currentPage: number;
  pageSize: number;
  filters: URLFilters;
  loading?: boolean;
  onCreateClick?: () => void;
  onUpdate?: (silent?: boolean) => void;
  onFilterChange: (filters: URLFilters) => void;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  isManualPageSize: boolean;
}

export function URLTable({
  urls,
  total,
  currentPage,
  pageSize,
  filters,
  loading = false, 
  onCreateClick, 
  onUpdate,
  onFilterChange,
  onPageChange,
  onPageSizeChange,
  isManualPageSize,
}: URLTableProps) {
  const router = useRouter();
  const { toast } = useToast();
  const [hoveredRow, setHoveredRow] = useState<string | null>(null);
  const [showFilters, setShowFilters] = useState(false);
  const [dateRange, setDateRange] = useState<{ from?: Date; to?: Date }>({});

  const totalPages = Math.ceil(total / pageSize);
  const hasActiveFilters = filters.is_active !== undefined || filters.created_after || filters.created_before;

  const handleRowClick = (urlId: string) => {
    router.push(`/?view=${urlId}`);
  };

  const handleStatusFilter = (value: string) => {
    const newFilters = { ...filters };
    if (value === 'all') {
      delete newFilters.is_active;
    } else {
      newFilters.is_active = value === 'active';
    }
    onFilterChange(newFilters);
  };

  const handleSortChange = (value: string) => {
    onFilterChange({ ...filters, sort_order: value as 'asc' | 'desc' });
  };

  const handleDateRangeApply = () => {
    const newFilters = { ...filters };
    if (dateRange.from) {
      newFilters.created_after = dateRange.from.toISOString();
    } else {
      delete newFilters.created_after;
    }
    if (dateRange.to) {
      // Set to end of day
      const endOfDay = new Date(dateRange.to);
      endOfDay.setHours(23, 59, 59, 999);
      newFilters.created_before = endOfDay.toISOString();
    } else {
      delete newFilters.created_before;
    }
    onFilterChange(newFilters);
  };

  const handleTimePreset = (hours?: number) => {
    if (hours === undefined) {
      // All time
      setDateRange({});
    } else {
      const now = new Date();
      const from = new Date(now.getTime() - hours * 60 * 60 * 1000);
      setDateRange({ from, to: now });
    }
    // Don't apply filters here - user must click Apply button
  };

  const handleClearFilters = () => {
    setDateRange({});
    onFilterChange({ sort_order: 'desc' });
  };

  const getCurrentStatusValue = () => {
    if (filters.is_active === undefined) return 'all';
    return filters.is_active ? 'active' : 'inactive';
  };

  if (loading && urls.length === 0) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Your URLs</CardTitle>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled
                className="h-8"
              >
                <Filter className="h-4 w-4 mr-2" />
                Filters
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Short URL</TableHead>
                <TableHead>Destination</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="-ml-3 h-8"
                    disabled
                  >
                    Created
                    <ArrowDown className="ml-2 h-4 w-4" />
                  </Button>
                </TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-32" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-48" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-5 w-16" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-20" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="h-8 w-32 ml-auto" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    );
  }

  if (urls.length === 0 && !hasActiveFilters) {
    return (
      <Card>
        <CardContent className="pt-6">
          <EmptyState
            icon={Link2}
            title="No URLs yet"
            description="Create your first short URL to get started. Share links that are easy to remember and track."
            action={
              onCreateClick
                ? {
                    label: "Create URL",
                    onClick: onCreateClick,
                  }
                : undefined
            }
          />
        </CardContent>
      </Card>
    );
  }

  const truncateUrl = (url: string, maxLength: number = 50) => {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + "...";
  };

  return (
    <Card className="flex flex-col h-full overflow-hidden">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Your URLs</CardTitle>
          <div className="flex items-center gap-2">
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClearFilters}
                className="h-8 text-xs"
              >
                <X className="h-3 w-3 mr-1" />
                Clear Filters
              </Button>
            )}
            <Popover open={showFilters} onOpenChange={setShowFilters}>
              <PopoverTrigger asChild>
                <Button
                  variant={showFilters ? "default" : "outline"}
                  size="sm"
                  className="h-8"
                >
                  <Filter className="h-4 w-4 mr-2" />
                  Filters
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-[600px] p-4" align="end" side="bottom">
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <h4 className="font-semibold text-sm">Filters</h4>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    {/* Status Filter */}
                    <div className="space-y-2">
                      <label className="text-xs font-medium">Status</label>
                      <Select value={getCurrentStatusValue()} onValueChange={handleStatusFilter}>
                        <SelectTrigger className="h-9 text-sm">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All</SelectItem>
                          <SelectItem value="active">Active</SelectItem>
                          <SelectItem value="inactive">Inactive</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    {/* Date Range Trigger */}
                    <div className="space-y-2">
                      <label className="text-xs font-medium">Date Range</label>
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button variant="outline" className="w-full h-9 justify-start text-left font-normal text-sm">
                            {dateRange.from ? (
                              dateRange.to ? (
                                <>
                                  {format(dateRange.from, "MMM dd")} - {format(dateRange.to, "MMM dd, yyyy")}
                                </>
                              ) : (
                                format(dateRange.from, "MMM dd, yyyy")
                              )
                            ) : (
                              <span className="text-muted-foreground text-xs">Pick a date range</span>
                            )}
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-auto p-0 flex flex-col" align="start">
                          {/* Quick Presets */}
                          <div className="p-3 space-y-2 border-b">
                            <div className="text-xs font-medium text-muted-foreground mb-2">Quick Filters</div>
                            <div className="grid grid-cols-2 gap-2">
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 text-xs"
                                onClick={() => handleTimePreset(1)}
                              >
                                Last Hour
                              </Button>
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 text-xs"
                                onClick={() => handleTimePreset(24)}
                              >
                                Last 24 Hours
                              </Button>
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 text-xs"
                                onClick={() => handleTimePreset(24 * 7)}
                              >
                                Last 7 Days
                              </Button>
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 text-xs"
                                onClick={() => handleTimePreset(24 * 30)}
                              >
                                Last 30 Days
                              </Button>
                            </div>
                            <Button
                              variant="outline"
                              size="sm"
                              className="w-full h-7 text-xs"
                              onClick={() => handleTimePreset(undefined)}
                            >
                              All Time
                            </Button>
                          </div>

                          {/* Custom Range Calendar */}
                          <div className="p-3 border-b">
                            <div className="text-xs font-medium text-muted-foreground mb-2">Custom Range</div>
                            <Calendar
                              mode="range"
                              selected={{ from: dateRange.from, to: dateRange.to }}
                              onSelect={(range) => {
                                setDateRange({ from: range?.from, to: range?.to });
                              }}
                              numberOfMonths={1}
                            />
                          </div>

                          <div className="flex items-center gap-2 p-3">
                            <Button
                              variant="outline"
                              size="sm"
                              className="flex-1 h-8"
                              onClick={() => {
                                setDateRange({});
                                const newFilters = { ...filters };
                                delete newFilters.created_after;
                                delete newFilters.created_before;
                                onFilterChange(newFilters);
                              }}
                            >
                              Clear
                            </Button>
                            <Button
                              size="sm"
                              className="flex-1 h-8"
                              onClick={handleDateRangeApply}
                              disabled={!dateRange.from && !dateRange.to}
                            >
                              Apply
                            </Button>
                          </div>
                        </PopoverContent>
                      </Popover>
                    </div>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col h-full overflow-hidden">
        {urls.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-muted-foreground">No URLs match your filters.</p>
            <Button variant="link" onClick={handleClearFilters} className="mt-2">
              Clear all filters
            </Button>
          </div>
        ) : (
          <>
            {/* Scrollable Table Container */}
            <div className="flex-1 overflow-auto relative" data-table-container>
              {loading && (
                <div className="absolute inset-0 bg-background/50 backdrop-blur-[1px] z-10 flex items-center justify-center">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                    Loading...
                  </div>
                </div>
              )}
              <Table>
                <TableHeader className="sticky top-0 z-10 bg-background">
                  <TableRow>
                    <TableHead>Short URL</TableHead>
                    <TableHead>Destination</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="-ml-3 h-8 data-[state=open]:bg-accent"
                        onClick={() => {
                          const newSortOrder = filters.sort_order === 'desc' ? 'asc' : 'desc';
                          handleSortChange(newSortOrder);
                        }}
                      >
                        Created
                        {filters.sort_order === 'desc' ? (
                          <ArrowDown className="ml-2 h-4 w-4" />
                        ) : filters.sort_order === 'asc' ? (
                          <ArrowUp className="ml-2 h-4 w-4" />
                        ) : (
                          <ArrowUpDown className="ml-2 h-4 w-4" />
                        )}
                      </Button>
                    </TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {urls.map((url) => (
                  <TableRow
                    key={url.id}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onMouseEnter={() => setHoveredRow(url.id)}
                    onMouseLeave={() => setHoveredRow(null)}
                    onClick={() => handleRowClick(url.id)}
                  >
                    <TableCell className="font-mono">
                      <div className="flex items-center gap-2">
                        <span 
                          className="text-sm font-medium cursor-pointer hover:text-primary transition-colors"
                          onClick={(e) => {
                            e.stopPropagation();
                            navigator.clipboard.writeText(url.short_url);
                            toast({
                              description: `Copied ${url.short_code} to clipboard!`,
                              duration: 2000,
                            });
                          }}
                          title="Click to copy"
                        >
                          {url.short_code}
                        </span>
                        <div className="w-7">
                          {hoveredRow === url.id && (
                            <CopyButton
                              value={url.short_url}
                              size="icon"
                              className="h-7 w-7"
                              successMessage={`Copied ${url.short_code} to clipboard!`}
                            />
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <div className="flex items-center gap-2 max-w-md">
                              <span className="text-sm text-muted-foreground truncate">
                                {truncateUrl(url.destination_url)}
                              </span>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6 flex-shrink-0"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  window.open(url.destination_url, "_blank");
                                }}
                              >
                                <ExternalLink className="h-3 w-3" />
                              </Button>
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-md">
                            <p className="break-all">{url.destination_url}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </TableCell>
                    <TableCell>
                      <Badge variant={url.is_active ? "default" : "secondary"} className="w-[70px] justify-center">
                        {url.is_active ? "Active" : "Inactive"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDistance(new Date(url.created_at), new Date(), {
                        addSuffix: true,
                      })}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRowClick(url.id);
                          }}
                        >
                          View Analytics
                        </Button>
                        <URLActions url={url} onUpdate={onUpdate} />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            </div>

            {/* Fixed Pagination - Always Visible */}
            <div className="flex-shrink-0 flex items-center justify-between mt-4 pt-4 border-t">
              <div className="flex items-center gap-4">
                <div className="text-sm text-muted-foreground">
                  Showing {urls.length > 0 ? currentPage * pageSize + 1 : 0} to {Math.min((currentPage + 1) * pageSize, total)} of {total} URLs
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground">Rows per page:</span>
                  <Select 
                    value={isManualPageSize ? pageSize.toString() : "auto"} 
                    onValueChange={(value) => {
                      if (value === "auto") {
                        onPageSizeChange(-1); // -1 signals to reset to auto mode
                      } else {
                        onPageSizeChange(Number(value));
                      }
                    }}
                  >
                    <SelectTrigger className="h-8 w-[80px]">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="auto">Auto</SelectItem>
                      <SelectItem value="10">10</SelectItem>
                      <SelectItem value="20">20</SelectItem>
                      <SelectItem value="50">50</SelectItem>
                      <SelectItem value="100">100</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              {totalPages > 1 && (
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      console.log('[URLTable] Previous clicked, currentPage:', currentPage);
                      onPageChange(currentPage - 1);
                    }}
                    disabled={currentPage === 0}
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Previous
                  </Button>
                  <div className="text-sm font-medium px-3">
                    Page {currentPage + 1} of {totalPages}
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      console.log('[URLTable] Next clicked, currentPage:', currentPage);
                      onPageChange(currentPage + 1);
                    }}
                    disabled={currentPage >= totalPages - 1}
                  >
                    Next
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
