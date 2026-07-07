import { memo, useState, useRef } from "react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import InfoBar from "./system/info-bar"
import { useSystemData } from "./system/use-system-data"
import { CpuChart, ContainerCpuChart } from "./system/charts/cpu-charts"
import { MemoryChart, ContainerMemoryChart, SwapChart } from "./system/charts/memory-charts"
import { RootDiskCharts, ExtraFsCharts } from "./system/charts/disk-charts"
import { BandwidthChart, ContainerNetworkChart } from "./system/charts/network-charts"
import { TemperatureChart, BatteryChart } from "./system/charts/sensor-charts"
import { LoadAverageChart } from "./system/charts/load-average-chart"
import { ContainerIcon, CpuIcon, HardDriveIcon } from "lucide-react"
import ContainersTable from "../containers-table/containers-table"

export default memo(function SystemDetail({ id }: { id: string }) {
	const systemData = useSystemData(id)

	const {
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
		maxValues,
		isLongerChart,
		showMax,
		dataEmpty,
		isPodman,
	} = systemData

	const [pageBottomExtraMargin, setPageBottomExtraMargin] = useState(0)
	const tabsRef = useRef<string[]>([])

	if (!system.id) {
		return null
	}

	const hasContainers = containerData.length > 0
	const tabs = ["core", "disk"]
	if (hasContainers) tabs.push("containers")
	tabsRef.current = tabs

	const coreProps = { chartData, grid, dataEmpty, showMax, isLongerChart, maxValues }

	function defaultLayout() {
		return (
			<>
				<div className="grid xl:grid-cols-2 gap-4">
					<CpuChart {...coreProps} />

					{hasContainers && (
						<ContainerCpuChart
							chartData={chartData}
							grid={grid}
							dataEmpty={dataEmpty}
							isPodman={isPodman}
							cpuConfig={containerChartConfigs.cpu}
						/>
					)}

					<MemoryChart {...coreProps} />

					{hasContainers && (
						<ContainerMemoryChart
							chartData={chartData}
							grid={grid}
							dataEmpty={dataEmpty}
							isPodman={isPodman}
							memoryConfig={containerChartConfigs.memory}
						/>
					)}

					<RootDiskCharts systemData={systemData} />

					<BandwidthChart {...coreProps} systemStats={systemStats} />

					{hasContainers && (
						<ContainerNetworkChart
							chartData={chartData}
							grid={grid}
							dataEmpty={dataEmpty}
							isPodman={isPodman}
							networkConfig={containerChartConfigs.network}
						/>
					)}

					<SwapChart chartData={chartData} grid={grid} dataEmpty={dataEmpty} systemStats={systemStats} />

					<LoadAverageChart chartData={chartData} grid={grid} dataEmpty={dataEmpty} />

					<TemperatureChart {...coreProps} />

					<BatteryChart {...coreProps} />
				</div>

				<ExtraFsCharts systemData={systemData} />

				{hasContainers && <ContainersTable systemId={system.id} />}
			</>
		)
	}

	function tabbedLayout() {
		return (
			<Tabs value={activeTab} onValueChange={setActiveTab} className="contents">
				<TabsList className="h-11 p-1.5 w-full shadow-xs overflow-auto justify-start">
					<TabsTrigger value="core" className="w-full flex items-center gap-1.5">
						<CpuIcon className="size-3.5" />
						<span>Core</span>
					</TabsTrigger>
					<TabsTrigger value="disk" className="w-full flex items-center gap-1.5">
						<HardDriveIcon className="size-3.5" />
						<span>Disk</span>
					</TabsTrigger>
					{hasContainers && (
						<TabsTrigger value="containers" className="w-full flex items-center gap-2">
							<ContainerIcon className="size-3.5" />
							<span>Containers</span>
						</TabsTrigger>
					)}
				</TabsList>

				<TabsContent value="core" forceMount className={activeTab === "core" ? "contents" : "hidden"}>
					<div className="grid xl:grid-cols-2 gap-4">
						<CpuChart {...coreProps} />
						<MemoryChart {...coreProps} />
						<LoadAverageChart chartData={chartData} grid={grid} dataEmpty={dataEmpty} />
						<BandwidthChart {...coreProps} systemStats={systemStats} />
						<TemperatureChart {...coreProps} setPageBottomExtraMargin={setPageBottomExtraMargin} />
						<BatteryChart {...coreProps} />
						<SwapChart chartData={chartData} grid={grid} dataEmpty={dataEmpty} systemStats={systemStats} />
						{pageBottomExtraMargin > 0 && <div style={{ marginBottom: pageBottomExtraMargin }}></div>}
					</div>
				</TabsContent>

				<TabsContent value="disk" forceMount className={activeTab === "disk" ? "contents" : "hidden"}>
					{mountedTabs.has("disk") && (
						<>
							<div className="grid xl:grid-cols-2 gap-4">
								<RootDiskCharts systemData={systemData} />
							</div>
							<ExtraFsCharts systemData={systemData} />
						</>
					)}
				</TabsContent>

				{hasContainers && (
					<TabsContent value="containers" forceMount className={activeTab === "containers" ? "contents" : "hidden"}>
						{mountedTabs.has("containers") && (
							<>
								<div className="grid xl:grid-cols-2 gap-4">
									<ContainerCpuChart
										chartData={chartData}
										grid={grid}
										dataEmpty={dataEmpty}
										isPodman={isPodman}
										cpuConfig={containerChartConfigs.cpu}
									/>
									<ContainerMemoryChart
										chartData={chartData}
										grid={grid}
										dataEmpty={dataEmpty}
										isPodman={isPodman}
										memoryConfig={containerChartConfigs.memory}
									/>
									<ContainerNetworkChart
										chartData={chartData}
										grid={grid}
										dataEmpty={dataEmpty}
										isPodman={isPodman}
										networkConfig={containerChartConfigs.network}
									/>
								</div>
								<ContainersTable systemId={system.id} />
							</>
						)}
					</TabsContent>
				)}
			</Tabs>
		)
	}

	return (
		<div className="grid gap-4 mb-14 overflow-x-clip">
			<InfoBar
				system={system}
				chartData={chartData}
				grid={grid}
				setGrid={setGrid}
				displayMode={displayMode}
				setDisplayMode={setDisplayMode}
				details={details}
			/>

			{displayMode === "tabs" ? tabbedLayout() : defaultLayout()}
		</div>
	)
})
