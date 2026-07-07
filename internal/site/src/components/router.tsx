import { createRouter } from "@nanostores/router"

const routes = {
	home: "/",
} as const

export const basePath = BESZEL?.BASE_PATH || ""

export const prependBasePath = (path: string) => (basePath + path).replaceAll("//", "/")

for (const route in routes) {
	// @ts-expect-error need as const above to get nanostores to parse types properly
	routes[route] = prependBasePath(routes[route])
}

export const $router = createRouter(routes, { links: false })

export const navigate = (urlString: string) => {
	$router.open(urlString)
}

export function Link(props: React.AnchorHTMLAttributes<HTMLAnchorElement>) {
	return (
		<a
			{...props}
			onClick={(e) => {
				e.preventDefault()
				const href = props.href || ""
				if (e.ctrlKey || e.metaKey) {
					window.open(href, "_blank")
				} else {
					navigate(href)
					props.onClick?.(e)
				}
			}}
		></a>
	)
}
