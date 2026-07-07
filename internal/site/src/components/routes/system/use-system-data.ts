import { useStore } from "@nanostores/react"
import { useEffect, useMemo, useRef, useState } from "react"
import { useContainerChartConfigs } from "@/components/charts/hooks"
import { pb } from "@/lib/api"
import { SystemStatus } from "@/lib/enums"
import {
	$chartTime,
	$currentSystem,
	$direction,
	$maxValues,
	$userSettings,
} from "@/lib/stores"
import { chartTimeData, listen, parseSemVer, useBrowserStorage } from "@/lib/utils"
import type {
	ChartData,
	ContainerStatsRecord,
	SystemDetailsRecord,
	SystemInfo,
	SystemRecord,
	SystemStats,
	SystemStatsRecord,
} from "@/types"
import { appendData, cache, getStats, getTimeData, makeContainerData, makeContainerPoint } from "./chart-data"

export type SystemData = ReturnType<typeof useSystemData>

export function useSystemData(id: string) {
	const direction = useStore($direction)
	const chartTime = useStore($chartTime)
	const maxValues = useStore($maxValues)
	const currentSystem = useStore($currentSystem)
	const [grid, setGrid] = useBrowserStorage("grid", true)
	const [displayMode, setDisplayMode] = useBrowserStorage<"default" | "tabs">("displayMode", "default")
	const [activeTab, setActiveTabRaw] = useState("core")
	const [mountedTabs, setMountedTabs] = useState(() => new Set<string>(["core"]))
	const tabsRef = useRef<string[]>(["core", "disk"])

	function setActiveTab(tab: string) {
		setActiveTabRaw(tab)
		setMountedTabs((prev) => (prev.has(tab) ? prev : new Set([...prev, tab])))
	}
	const [system, setSystem] = useState({} as SystemRecord)
	const [systemStats, setSystemStats] = useState([] as SystemStatsRecord[])
	const [containerData, setContainerData] = useState([] as ChartData["containerData"])
	const statsRequestId = useRef(0)
	const [chartLoading, setChartLoading] = useState(true)
	const [details, setDetails] = useState<SystemDetailsRecord>({} as SystemDetailsRecord)

	useEffect(() => {
		return () => {
			setSystemStats([])
			setContainerData([])
			setDetails({} as SystemDetailsRecord)
		}
	}, [id])

	// use single current system
	useEffect(() => {
		if (currentSystem) {
			setSystem(currentSystem)
			document.title = `${currentSystem.name} / HomeMonit`
		}
	}, [currentSystem])

	// hide 1m chart time if system agent version is less than 0.13.0
	useEffect(() => {
		if (system?.info?.v && parseSemVer(system.info.v) < parseSemVer("0.13.0")) {
			$chartTime.set("1h")
		}
	}, [system?.info?.v])

	// fetch system details
	useEffect(() => {
		if (!system.id || system.info?.m) {
			return
		}
		pb.collection<SystemDetailsRecord>("system_details")
			.getOne(system.id, {
				fields: "hostname,kernel,cores,threads,cpu,os,os_name,arch,memory,podman",
				headers: {
					"Cache-Control": "public, max-age=60",
				},
			})
			.then(setDetails)
	}, [system.id])

	// subscribe to realtime metrics if chart time is 1m
	useEffect(() => {
		let unsub = () => {}
		if (!system.id || chartTime !== "1m") {
			return
		}
		if (system.status !== SystemStatus.Up || parseSemVer(system?.info?.v).minor < 13) {
			$chartTime.set("1h")
			return
		}
		let isFirst = true
		pb.realtime
			.subscribe(
				`rt_metrics`,
				(data: { container: ContainerStatsRecord[]; info: SystemInfo; stats: SystemStats }) => {
					const now = Date.now()
					const statsPoint = { created: now, stats: data.stats } as SystemStatsRecord
					const containerPoint =
						data.container?.length > 0
							? makeContainerPoint(now, data.container as unknown as ContainerStatsRecord["stats"])
							: null
					if (isFirst) {
						isFirst = false
						setSystemStats([statsPoint])
						setContainerData(containerPoint ? [containerPoint] : [])
						return
					}
					setSystemStats((prev) => appendData(prev, [statsPoint], 1000, 60))
					if (containerPoint) {
						setContainerData((prev) => appendData(prev, [containerPoint], 1000, 60))
					}
				},
				{ query: { system: system.id } }
			)
			.then((us) => {
				unsub = us
			})
		return () => {
			unsub?.()
		}
	}, [chartTime, system.id])

	const agentVersion = useMemo(() => parseSemVer(system?.info?.v), [system?.info?.v])

	const chartData: ChartData = useMemo(() => {
		const lastCreated = Math.max(
			(systemStats.at(-1)?.created as number) ?? 0,
			(containerData.at(-1)?.created as number) ?? 0
		)
		return {
			systemStats,
			containerData,
			chartTime,
			orientation: direction === "rtl" ? "right" : "left",
			...getTimeData(chartTime, lastCreated),
			agentVersion,
		}
	}, [systemStats, containerData, direction])

	const containerChartConfigs = useContainerChartConfigs(containerData)

	// get stats
	useEffect(() => {
		if (!system.id || !chartTime || chartTime === "1m") {
			return
		}

		const systemId = system.id
		const { expectedInterval } = chartTimeData[chartTime]
		const ss_cache_key = `${systemId}_${chartTime}_system_stats`
		const cs_cache_key = `${systemId}_${chartTime}_container_stats`
		const requestId = ++statsRequestId.current

		const cachedSystemStats = cache.get(ss_cache_key) as SystemStatsRecord[] | undefined
		const cachedContainerData = cache.get(cs_cache_key) as ChartData["containerData"] | undefined

		if (cachedSystemStats?.length) {
			setSystemStats(cachedSystemStats)
			setContainerData(cachedContainerData || [])
			setChartLoading(false)

			const lastCreated = cachedSystemStats.at(-1)?.created as number | undefined
			if (lastCreated && Date.now() - lastCreated < expectedInterval * 0.9) {
				return
			}
		} else {
			setChartLoading(true)
		}

		Promise.allSettled([
			getStats<SystemStatsRecord>("system_stats", systemId, chartTime),
			getStats<ContainerStatsRecord>("container_stats", systemId, chartTime),
		]).then(([systemStats, containerStats]) => {
			if (requestId !== statsRequestId.current) {
				return
			}

			setChartLoading(false)

			let systemData = (cache.get(ss_cache_key) || []) as SystemStatsRecord[]
			if (systemStats.status === "fulfilled" && systemStats.value.length) {
				systemData = appendData(systemData, systemStats.value, expectedInterval, 100)
				cache.set(ss_cache_key, systemData)
			}
			setSystemStats(systemData)

			let containerData = (cache.get(cs_cache_key) || []) as ChartData["containerData"]
			if (containerStats.status === "fulfilled" && containerStats.value.length) {
				containerData = appendData(containerData, makeContainerData(containerStats.value), expectedInterval, 100)
				cache.set(cs_cache_key, containerData)
			}
			setContainerData(containerData)
		})
	}, [system, chartTime])

	// arrow keys switch tabs if in tabs mode
	useEffect(() => {
		const handleKeyUp = (e: KeyboardEvent) => {
			if (
				e.target instanceof HTMLInputElement ||
				e.target instanceof HTMLTextAreaElement ||
				e.ctrlKey ||
				e.metaKey ||
				e.altKey
			) {
				return
			}

			const isLeft = e.key === "ArrowLeft" || e.key === "h"
			const isRight = e.key === "ArrowRight" || e.key === "l"
			if (!isLeft && !isRight) {
				return
			}

			if (displayMode === "tabs") {
				if (!e.shiftKey) {
					if (e.target instanceof HTMLElement && e.target.closest('[role="tablist"]')) {
						return
					}
					const tabs = tabsRef.current
					const currentIdx = tabs.indexOf(activeTab)
					const nextIdx = isLeft ? (currentIdx - 1 + tabs.length) % tabs.length : (currentIdx + 1) % tabs.length
					setActiveTab(tabs[nextIdx])
					return
				}
			}
		}
		return listen(document, "keyup", handleKeyUp)
	}, [displayMode, activeTab])

	const isLongerChart = !["1m", "1h"].includes(chartTime)
	const showMax = maxValues && isLongerChart
	const dataEmpty = !chartLoading && chartData.systemStats.length === 0
	const isPodman = details?.podman ?? system.info?.p ?? false

	return {
		system,
		systemStats,
		containerData,
		chartData,
		containerChartConfigs,
		details,
		grid,
		setGrid,
		displayMode,
		setDisplayMode,
		activeTab,
		setActiveTab,
		mountedTabs,
		tabsRef,
		maxValues,
		isLongerChart,
		showMax,
		dataEmpty,
		isPodman,
	}
}
