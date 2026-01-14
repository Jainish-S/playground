const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8000';

interface FetchOptions extends RequestInit {
  token?: string;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async fetch<T>(endpoint: string, options: FetchOptions = {}): Promise<T> {
    const { token, ...fetchOptions } = options;
    
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...fetchOptions,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  }

  // URL endpoints
  async createURL(data: CreateURLRequest, token: string) {
    return this.fetch<URLResponse>('/v1/urls', {
      method: 'POST',
      body: JSON.stringify(data),
      token,
    });
  }

  async listURLs(
    token: string, 
    limit = 20, 
    offset = 0,
    filters?: {
      is_active?: boolean;
      sort_order?: 'asc' | 'desc';
      created_after?: string;
      created_before?: string;
    }
  ) {
    let queryParams = `limit=${limit}&offset=${offset}`;
    
    if (filters) {
      if (filters.is_active !== undefined) {
        queryParams += `&is_active=${filters.is_active}`;
      }
      if (filters.sort_order) {
        queryParams += `&sort_order=${filters.sort_order}`;
      }
      if (filters.created_after) {
        queryParams += `&created_after=${encodeURIComponent(filters.created_after)}`;
      }
      if (filters.created_before) {
        queryParams += `&created_before=${encodeURIComponent(filters.created_before)}`;
      }
    }
    
    console.log('[API] Calling /v1/urls with params:', queryParams, 'filters:', filters);
    return this.fetch<URLListResponse>(`/v1/urls?${queryParams}`, { token });
  }

  async getURL(id: string, token: string) {
    return this.fetch<URLResponse>(`/v1/urls/${id}`, { token });
  }

  async updateURL(id: string, data: UpdateURLRequest, token: string) {
    return this.fetch<URLResponse>(`/v1/urls/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
      token,
    });
  }

  async deleteURL(id: string, token: string) {
    return this.fetch<void>(`/v1/urls/${id}`, { method: 'DELETE', token });
  }

  async checkCode(code: string, token: string) {
    return this.fetch<{ available: boolean; reason?: string }>(`/v1/urls/check/${code}`, { token });
  }

  // Analytics endpoints
  async getAnalytics(urlId: string, token: string) {
    return this.fetch<AnalyticsResponse>(`/v1/urls/${urlId}/analytics`, { token });
  }

  async getClicksOverTime(urlId: string, token: string, days = 7) {
    return this.fetch<ClicksTimeSeriesResponse>(`/v1/urls/${urlId}/analytics/clicks?days=${days}`, { token });
  }

  async getGeoBreakdown(urlId: string, token: string) {
    return this.fetch<GeoBreakdownResponse>(`/v1/urls/${urlId}/analytics/geo`, { token });
  }

  async getDeviceBreakdown(urlId: string, token: string) {
    return this.fetch<DeviceBreakdownResponse>(`/v1/urls/${urlId}/analytics/devices`, { token });
  }

  async getDashboard(token: string) {
    return this.fetch<DashboardResponse>('/v1/dashboard', { token });
  }
}

// Types
export interface CreateURLRequest {
  destination_url: string;
  custom_code?: string;
  notes?: string;
  expires_in?: number;
}

export interface UpdateURLRequest {
  destination_url?: string;
  notes?: string;
  expires_in?: number;
  is_active?: boolean;
}

export interface URLResponse {
  id: string;
  short_code: string;
  short_url: string;
  destination_url: string;
  notes?: string;
  is_active: boolean;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface URLListResponse {
  urls: URLResponse[];
  total: number;
  limit: number;
  offset: number;
}

export interface AnalyticsResponse {
  url_id: string;
  short_code: string;
  total_clicks: number;
  unique_visitors: number;
  mobile_clicks: number;
  desktop_clicks: number;
  tablet_clicks: number;
  devices: { device_type: string; clicks: number }[];
  browsers: { browser: string; clicks: number }[];
}

export interface ClicksTimeSeriesResponse {
  url_id: string;
  days: number;
  data: { bucket: string; clicks: number; unique: number }[];
}

export interface GeoBreakdownResponse {
  url_id: string;
  data: { country: string; clicks: number }[];
}

export interface DeviceBreakdownResponse {
  url_id: string;
  devices: { device_type: string; clicks: number }[];
  browsers: { browser: string; clicks: number }[];
}

export interface DashboardResponse {
  total_urls: number;
  total_clicks: number;
  unique_visitors: number;
}

export const apiClient = new ApiClient(API_BASE_URL);
