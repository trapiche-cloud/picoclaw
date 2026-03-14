import { Outlet, createRootRoute, useNavigate, useRouterState } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"
import { useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { Toaster } from "sonner"

import { getAuthStatus } from "@/api/auth"
import { AppLayout } from "@/components/app-layout"

const RootLayout = () => {
  const navigate = useNavigate()
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  const { data: authStatus, isLoading } = useQuery({
    queryKey: ["authStatus"],
    queryFn: getAuthStatus,
    retry: false,
    staleTime: 30_000,
  })

  useEffect(() => {
    if (isLoading || !authStatus) return

    if (authStatus.auth_enabled) {
      if (!authStatus.authenticated && currentPath !== "/login") {
        navigate({ to: "/login" })
      } else if (authStatus.authenticated && currentPath === "/login") {
        navigate({ to: "/" })
      }
    } else if (currentPath === "/login") {
      navigate({ to: "/" })
    }
  }, [authStatus, isLoading, currentPath, navigate])

  if (isLoading) {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-background">
        <div className="text-muted-foreground text-sm">Loading...</div>
      </div>
    )
  }

  // Login page renders without AppLayout
  if (currentPath === "/login") {
    return (
      <>
        <Outlet />
        <Toaster position="bottom-center" />
        <TanStackRouterDevtools />
      </>
    )
  }

  return (
    <AppLayout>
      <Outlet />
      <TanStackRouterDevtools />
    </AppLayout>
  )
}

export const Route = createRootRoute({ component: RootLayout })
