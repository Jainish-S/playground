"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { StatsCards } from "@/components/dashboard/StatsCards";
import { URLTable } from "@/components/dashboard/URLTable";
import { CreateURLButton } from "@/components/dashboard/CreateURLButton";
import { CreateURLModal } from "@/components/dashboard/CreateURLModal";
import { URLDetailsSheet } from "@/components/dashboard/URLDetailsSheet";
import { DashboardResponse, URLListResponse } from "@/lib/api";
import { getDashboardData, listURLsAction } from "@/app/actions";

export interface URLFilters {
  is_active?: boolean;
  sort_order?: 'asc' | 'desc';
  created_after?: string;
  created_before?: string;
}

function DashboardContentInner() {
  const searchParams = useSearchParams();
  const viewUrlId = searchParams.get("view");

  const [stats, setStats] = useState<DashboardResponse | null>(null);
  const [urlData, setUrlData] = useState<URLListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [statsLoading, setStatsLoading] = useState(true);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [filters, setFilters] = useState<URLFilters>({ sort_order: 'desc' });
  const [currentPage, setCurrentPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [isManualPageSize, setIsManualPageSize] = useState(false); // Track if user manually selected
  const [isInitialized, setIsInitialized] = useState(false); // Track if dynamic size has been calculated

  // Fetch dashboard stats only on initial load
  useEffect(() => {
    const loadStats = async () => {
      try {
        setStatsLoading(true);
        const dashboardResult = await getDashboardData();
        if (dashboardResult.success && dashboardResult.data) {
          setStats(dashboardResult.data);
        }
      } catch (error) {
        console.error("Failed to fetch dashboard stats:", error);
      } finally {
        setStatsLoading(false);
      }
    };

    loadStats();
  }, []);

  // Fetch URLs whenever filters or pagination change
  useEffect(() => {
    // Skip the first call until dynamic page size has been initialized
    if (!isInitialized) {
      console.log('[UseEffect] Skipping initial URL fetch until page size is calculated');
      return;
    }

    const loadUrls = async () => {
      try {
        setLoading(true);
        console.log('[UseEffect] Fetching URLs with currentPage:', currentPage, 'offset:', currentPage * pageSize);
        const urlsResult = await listURLsAction(pageSize, currentPage * pageSize, filters);
        if (urlsResult.success && urlsResult.data) {
          setUrlData(urlsResult.data);
        }
      } catch (error) {
        console.error("Failed to fetch URLs:", error);
      } finally {
        setLoading(false);
      }
    };

    loadUrls();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage, pageSize, JSON.stringify(filters), isInitialized]);

  const fetchData = async (silent = false) => {
    try {
      if (!silent) {
        setLoading(true);
      }

      const [dashboardResult, urlsResult] = await Promise.all([
        getDashboardData(),
        listURLsAction(pageSize, currentPage * pageSize, filters),
      ]);

      if (dashboardResult.success && dashboardResult.data) {
        setStats(dashboardResult.data);
      }
      if (urlsResult.success && urlsResult.data) {
        setUrlData(urlsResult.data);
      }
    } catch (error) {
      console.error("Failed to fetch dashboard data:", error);
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  };

  const handleFilterChange = (newFilters: URLFilters) => {
    setFilters(newFilters);
    setCurrentPage(0); // Reset to first page when filters change
  };

  const handlePageChange = (page: number) => {
    console.log('[Pagination] Changing page from', currentPage, 'to', page);
    setCurrentPage(page);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    if (newPageSize === -1) {
      // -1 signals reset to auto mode
      setIsManualPageSize(false);
      // Don't set pageSize here, let the auto-calculation handle it
      setCurrentPage(0);
    } else {
      setPageSize(newPageSize);
      setCurrentPage(0); // Reset to first page when page size changes
      setIsManualPageSize(true); // Disable auto-calculation after manual selection
    }
  };

  // Dynamic page size calculation based on available height
  useEffect(() => {
    // Skip if manually selected or already initialized
    if (isManualPageSize || isInitialized) return;

    // Calculate immediately from viewport dimensions
    const windowHeight = window.innerHeight;
    const estimatedOverhead = 360; // header + stats + filters + pagination + padding
    const availableHeight = Math.max(200, windowHeight - estimatedOverhead);
    const estimatedRowHeight = 53;
    const optimalRows = Math.floor(availableHeight / estimatedRowHeight);
    const clampedRows = Math.max(5, Math.min(50, optimalRows));
    
    console.log('[Dynamic PageSize] Calculated from viewport:', {
      windowHeight,
      availableHeight,
      optimalRows: clampedRows
    });
    
    if (clampedRows !== pageSize) {
      setPageSize(clampedRows);
      setCurrentPage(0);
    }
    setIsInitialized(true);

    // No async  logic needed - we calculated synchronously above

    // No cleanup needed for this effect
  }, [isManualPageSize, isInitialized, pageSize]);

  // Separate effect for dynamic resizing after container is mounted
  useEffect(() => {
    if (isManualPageSize || !isInitialized) return;

    const handleResize = () => {
      const tableContainer = document.querySelector('[data-table-container]');
      if (!tableContainer) return;

      const tableRow = tableContainer.querySelector('tbody tr');
      if (!tableRow) return;

      const rowHeight = tableRow.getBoundingClientRect().height;
      const containerHeight = tableContainer.getBoundingClientRect().height;
      const availableHeight = containerHeight - 10;
      const optimalRows = Math.floor(availableHeight / rowHeight);
      const clampedRows = Math.max(5, Math.min(50, optimalRows));

      if (clampedRows !== pageSize) {
        console.log('[Dynamic PageSize] Recalculated after resize:', clampedRows);
        setPageSize(clampedRows);
        setCurrentPage(0);
      }
    };

    const resizeObserver = new ResizeObserver(handleResize);
    const tableContainer = document.querySelector('[data-table-container]');
    if (tableContainer) {
      resizeObserver.observe(tableContainer);
    }

    return () => {
      resizeObserver.disconnect();
    };
  }, [isManualPageSize, isInitialized, pageSize]);

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Stats Section */}
      <div className="flex-shrink-0 pb-6">
        <StatsCards stats={stats} loading={statsLoading} />
      </div>

      {/* Scrollable Table Section */}
      <div className="flex-1 overflow-hidden flex flex-col">
        <URLTable
          urls={urlData?.urls || []}
          total={urlData?.total || 0}
          currentPage={currentPage}
          pageSize={pageSize}
          isManualPageSize={isManualPageSize}
          filters={filters}
          loading={loading}
          onCreateClick={() => setCreateModalOpen(true)}
          onUpdate={(silent) => fetchData(silent)}
          onFilterChange={handleFilterChange}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      </div>

      <CreateURLButton onClick={() => setCreateModalOpen(true)} />

      <CreateURLModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onSuccess={() => fetchData()}
      />

      <URLDetailsSheet urlId={viewUrlId} />
    </div>
  );
}

export default function DashboardContent() {
  return (
    <Suspense fallback={<div className="p-8">Loading dashboard...</div>}>
      <DashboardContentInner />
    </Suspense>
  );
}
