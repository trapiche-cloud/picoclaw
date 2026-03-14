const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message || `API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export interface AuthStatus {
  authenticated: boolean
  auth_enabled: boolean
}

export async function getAuthStatus(): Promise<AuthStatus> {
  return request<AuthStatus>("/api/auth/status")
}

export async function login(
  username: string,
  password: string,
): Promise<{ authenticated: boolean }> {
  return request<{ authenticated: boolean }>("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  })
}

export async function logout(): Promise<{ authenticated: boolean }> {
  return request<{ authenticated: boolean }>("/api/auth/logout", {
    method: "POST",
  })
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<{ updated: boolean }> {
  return request<{ updated: boolean }>("/api/auth/password", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  })
}
