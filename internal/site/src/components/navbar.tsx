import { LogOutIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Link } from "./router"
import { logOut, pb } from "@/lib/api"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export default function Navbar() {
	return (
		<div className="flex items-center h-14 md:h-16 bg-card px-4 pe-3 sm:px-6 border border-border/60 bt-0 rounded-md my-4">
			<Link
				href="/"
				aria-label="Home"
				className="p-2 ps-0 me-3 flex items-center gap-2 group font-semibold text-lg"
			>
				<span className="bg-primary text-primary-foreground size-7 rounded flex items-center justify-center font-bold text-base">H</span>
				<span>HomeMonit</span>
			</Link>

			{/* desktop nav */}
			<div className="flex items-center ms-auto gap-2">
				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="ghost" className="relative h-8 w-8 rounded-full bg-muted flex items-center justify-center">
							<span className="text-xs font-semibold uppercase">{pb.authStore.record?.email?.[0] ?? "U"}</span>
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end" className="min-w-44">
						<DropdownMenuLabel className="font-normal">
							<div className="flex flex-col space-y-1">
								<p className="text-sm font-medium leading-none">Administrator</p>
								<p className="text-xs leading-none text-muted-foreground">{pb.authStore.record?.email}</p>
							</div>
						</DropdownMenuLabel>
						<DropdownMenuSeparator />
						<DropdownMenuItem onSelect={logOut}>
							<LogOutIcon className="me-2.5 h-4 w-4" />
							<span>Log Out</span>
						</DropdownMenuItem>
					</DropdownMenuContent>
				</DropdownMenu>
			</div>
		</div>
	)
}
