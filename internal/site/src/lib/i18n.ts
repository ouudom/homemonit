import { i18n } from "@lingui/core"

export function getLocale() {
	return "en"
}

export async function dynamicActivate(locale: string) {
	i18n.activate(locale)
}
