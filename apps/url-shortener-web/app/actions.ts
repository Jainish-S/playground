"use server";

import { auth0 } from "@/lib/auth0";
import { apiClient } from "@/lib/api";

export async function getDashboardData() {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.getDashboard(accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function listURLsAction(
  limit: number = 10, 
  offset: number = 0,
  filters?: {
    is_active?: boolean;
    sort_order?: 'asc' | 'desc';
    created_after?: string;
    created_before?: string;
  }
) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.listURLs(accessToken.token, limit, offset, filters);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function createURLAction(data: {
  destination_url: string;
  custom_code?: string;
  notes?: string;
}) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.createURL(data, accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function updateURLAction(
  id: string,
  data: {
    destination_url?: string;
    is_active?: boolean;
    notes?: string;
  }
) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.updateURL(id, data, accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function deleteURLAction(id: string) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    await apiClient.deleteURL(id, accessToken.token);
    return { success: true };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function getURLAction(id: string) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error:  "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.getURL(id, accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function checkCodeAction(code: string) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.checkCode(code, accessToken.token);
    return { success: true, available: result.available };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

// Analytics actions
export async function getAnalyticsAction(urlId: string) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.getAnalytics(urlId, accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function getClicksTimeSeriesAction(urlId: string, days: number = 7) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.getClicksOverTime(urlId, accessToken.token, days);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function getGeoBreakdownAction(urlId: string) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.getGeoBreakdown(urlId, accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}

export async function getDeviceBreakdownAction(urlId: string) {
  try {
    const session = await auth0.getSession();
    if (!session) {
      return { success: false, error: "Unauthorized" };
    }

    const accessToken = await auth0.getAccessToken();
    if (!accessToken) {
      return { success: false, error: "No access token" };
    }

    const result = await apiClient.getDeviceBreakdown(urlId, accessToken.token);
    return { success: true, data: result };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
}
