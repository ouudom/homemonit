import { atom, listenKeys, map } from "nanostores"
import type { ChartTimes, SystemRecord, UserSettings } from "@/types"
import { pb } from "./api"
import { Unit } from "./enums"

/** Default layout width. Used as fallback when user setting is unset. */
export const defaultLayoutWidth = 1580

/** Store if user is authenticated */
export const $authenticated = atom(pb.authStore.isValid)

/** Single system record for HomeMonit */
export const $currentSystem = atom<SystemRecord | null>(null)

/** SSH public key */
export const $publicKey = atom("")

/** Chart time period */
export const $chartTime = atom<ChartTimes>("1h")

/** Whether to display average or max chart values */
export const $maxValues = atom(false)

/** User settings */
export const $userSettings = map<UserSettings>({
	chartTime: "1h",
	emails: [pb.authStore.record?.email || ""],
	unitNet: Unit.Bytes,
	unitTemp: Unit.Celsius,
})
// update chart time on change
listenKeys($userSettings, ["chartTime"], ({ chartTime }) => $chartTime.set(chartTime))

/** Temperature chart filter */
export const $temperatureFilter = atom("")

/** Container chart filter */
export const $containerFilter = atom("")

/** Direction for localization */
export const $direction = atom<"ltr" | "rtl">("ltr")
