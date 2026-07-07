import { useEffect, useState } from "react"
import { UserAuthForm } from "@/components/login/auth-form"
import { pb } from "@/lib/api"
import { useTheme } from "../theme-provider"

export default function Login() {
	const [isFirstRun, setFirstRun] = useState(false)
	const { theme } = useTheme()

	useEffect(() => {
		document.title = "Login / HomeMonit"

		pb.send("/api/beszel/first-run", {}).then(({ firstRun }) => {
			setFirstRun(firstRun)
		})
	}, [])

	const subtitle = isFirstRun
		? "Please create an admin account"
		: "Please sign in to your account"

	return (
		<div className="min-h-svh grid items-center py-12">
			<div
				className="grid gap-5 w-full px-4 mx-auto"
				// @ts-expect-error
				style={{ maxWidth: "21.5em", "--border": theme == "light" ? "hsl(30, 8%, 70%)" : "hsl(220, 3%, 25%)" }}
			>
				<div className="text-center">
					<div className="mb-4 flex items-center justify-center gap-2 font-bold text-2xl text-foreground">
						<span className="bg-primary text-primary-foreground size-8 rounded flex items-center justify-center text-lg">H</span>
						<span>HomeMonit</span>
					</div>
					<p className="text-sm text-muted-foreground">{subtitle}</p>
				</div>
				<UserAuthForm isFirstRun={isFirstRun} />
			</div>
		</div>
	)
}
