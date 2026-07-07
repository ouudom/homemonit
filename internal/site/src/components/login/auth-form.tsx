import { KeyIcon, LoaderCircle, LockIcon, LogInIcon, MailIcon } from "lucide-react"
import { useCallback, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { pb } from "@/lib/api"
import { $authenticated } from "@/lib/stores"
import { cn } from "@/lib/utils"
import { toast } from "../ui/use-toast"

export const showLoginFaliedToast = (description = "Please check your credentials and try again") => {
	toast({
		title: "Login attempt failed",
		description,
		variant: "destructive",
	})
}

export function UserAuthForm({
	className,
	isFirstRun,
	...props
}: {
	className?: string
	isFirstRun: boolean
}) {
	const [isLoading, setIsLoading] = useState<boolean>(false)
	const [errors, setErrors] = useState<Record<string, string | undefined>>({})

	const handleSubmit = useCallback(
		async (e: React.FormEvent<HTMLFormElement>) => {
			e.preventDefault()
			setIsLoading(true)
			setErrors({})

			try {
				const formData = new FormData(e.target as HTMLFormElement)
				const email = formData.get("email") as string
				const password = formData.get("password") as string

				if (!email || !password) {
					setErrors({ email: "Email and password are required." })
					return
				}

				if (isFirstRun) {
					const passwordConfirm = formData.get("passwordConfirm") as string
					if (password !== passwordConfirm) {
						setErrors({ passwordConfirm: "Passwords do not match." })
						return
					}
					await pb.send("/api/beszel/create-user", {
						method: "POST",
						body: JSON.stringify({ email, password }),
					})
				}

				await pb.collection("users").authWithPassword(email, password)
				$authenticated.set(true)
			} catch (err: any) {
				showLoginFaliedToast(err?.message || "Invalid credentials.")
			} finally {
				setIsLoading(false)
			}
		},
		[isFirstRun]
	)

	return (
		<div className={cn("grid gap-6", className)} {...props}>
			<form onSubmit={handleSubmit} onChange={() => setErrors({})}>
				<div className="grid gap-2.5">
					<div className="grid gap-1 relative">
						<MailIcon className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
						<Label className="sr-only" htmlFor="email">
							Email
						</Label>
						<Input
							id="email"
							name="email"
							required
							placeholder="name@example.com"
							type="email"
							autoCapitalize="none"
							autoComplete="email"
							autoCorrect="off"
							disabled={isLoading}
							className={cn("ps-9", errors?.email && "border-red-500")}
						/>
						{errors?.email && <p className="px-1 text-xs text-red-600">{errors.email}</p>}
					</div>
					<div className="grid gap-1 relative">
						<LockIcon className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
						<Label className="sr-only" htmlFor="password">
							Password
						</Label>
						<Input
							id="password"
							name="password"
							required
							placeholder="Password"
							type="password"
							autoCapitalize="none"
							autoComplete="current-password"
							autoCorrect="off"
							disabled={isLoading}
							className={cn("ps-9", errors?.password && "border-red-500")}
						/>
						{errors?.password && <p className="px-1 text-xs text-red-600">{errors.password}</p>}
					</div>

					{isFirstRun && (
						<div className="grid gap-1 relative">
							<KeyIcon className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
							<Label className="sr-only" htmlFor="passwordConfirm">
								Confirm Password
							</Label>
							<Input
								id="passwordConfirm"
								name="passwordConfirm"
								required
								placeholder="Confirm Password"
								type="password"
								autoCapitalize="none"
								autoComplete="new-password"
								autoCorrect="off"
								disabled={isLoading}
								className={cn("ps-9", errors?.passwordConfirm && "border-red-500")}
							/>
							{errors?.passwordConfirm && (
								<p className="px-1 text-xs text-red-600">{errors.passwordConfirm}</p>
							)}
						</div>
					)}

					<Button disabled={isLoading} className="mt-1">
						{isLoading ? (
							<LoaderCircle className="me-2 h-4 w-4 animate-spin" />
						) : (
							<LogInIcon className="me-2 h-4 w-4" />
						)}
						{isFirstRun ? "Create Account" : "Sign In"}
					</Button>
				</div>
			</form>
		</div>
	)
}
