import "./index.css"
import { i18n } from "@lingui/core"
import { I18nProvider } from "@lingui/react"
import { useStore } from "@nanostores/react"
import { DirectionProvider } from "@radix-ui/react-direction"
import { lazy, memo, Suspense, useEffect } from "react"
import ReactDOM from "react-dom/client"
import Navbar from "@/components/navbar.tsx"
import { $router } from "@/components/router.tsx"
import { ThemeProvider } from "@/components/theme-provider.tsx"
import { Toaster } from "@/components/ui/toaster.tsx"
import { pb } from "@/lib/api.ts"
import { dynamicActivate, getLocale } from "@/lib/i18n"
import {
	$authenticated,
	$currentSystem,
	$direction,
	$publicKey,
	defaultLayoutWidth,
} from "@/lib/stores.ts"
import type { BeszelInfo } from "./types"

const LoginPage = lazy(() => import("@/components/login/login.tsx"))
const SystemDetail = lazy(() => import("@/components/routes/system.tsx"))

const App = memo(() => {
	const currentSystem = useStore($currentSystem)

	useEffect(() => {
		const fetchSystem = () => {
			pb.collection("systems")
				.getFirstListItem("")
				.then((sys) => {
					$currentSystem.set(sys)
				})
				.catch((err) => {
					console.log("No system found yet:", err)
				})
		}

		let unsubscribeRealtime = () => {}

		if (pb.authStore.isValid) {
			fetchSystem()
			pb.collection("systems")
				.subscribe("*", (e) => {
					const current = $currentSystem.get()
					if (current && e.record.id === current.id) {
						$currentSystem.set(e.record)
					} else if (!current && e.action === "create") {
						$currentSystem.set(e.record)
					}
				})
				.then((unsub) => {
					unsubscribeRealtime = unsub
				})
		}

		// get general info for authenticated users
		pb.send<BeszelInfo>("/api/beszel/info", {}).then((data) => {
			$publicKey.set(data.key)
		})

		return () => {
			unsubscribeRealtime()
		}
	}, [])

	if (!currentSystem) {
		return (
			<div className="flex flex-col items-center justify-center min-h-[50vh] text-center p-6 bg-card rounded-lg border">
				<h2 className="text-xl font-semibold mb-2">Connecting to Homeserver...</h2>
				<p className="text-muted-foreground text-sm max-w-md">
					Make sure your homemonit-agent is running and configured with the correct token.
				</p>
			</div>
		)
	}

	return <SystemDetail id={currentSystem.id} />
})

const Layout = () => {
	const authenticated = useStore($authenticated)
	const direction = useStore($direction)

	useEffect(() => {
		document.documentElement.dir = direction
	}, [direction])

	return (
		<DirectionProvider dir={direction}>
			{!authenticated ? (
				<Suspense>
					<LoginPage />
				</Suspense>
			) : (
				<div style={{ "--container": `${defaultLayoutWidth}px` } as React.CSSProperties}>
					<div className="container">
						<Navbar />
					</div>
					<div className="container relative">
						<Suspense fallback={
							<div className="flex items-center justify-center min-h-[30vh]">
								<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
							</div>
						}>
							<App />
						</Suspense>
					</div>
				</div>
			)}
		</DirectionProvider>
	)
}

const I18nApp = () => {
	useEffect(() => {
		dynamicActivate(getLocale())
	}, [])

	return (
		<I18nProvider i18n={i18n}>
			<ThemeProvider>
				<Layout />
				<Toaster />
			</ThemeProvider>
		</I18nProvider>
	)
}

ReactDOM.createRoot(document.getElementById("app") as HTMLElement).render(
	<I18nApp />
)
