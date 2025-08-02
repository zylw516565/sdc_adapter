SDC_Adapter is a vehicle-end CAN data acquisition and parsing subsystem developed with Golang.

SDC_Adapter 是一套用Golang开发的车端CAN数据采集,解析子系统。SDC是灵活数据采集(Smart Data Collection)的简称。
运行在车端CCM主控模块(arm架构芯片),从车端CCM的字符设备上捞取CAN数据,解析并处理。
该系统支持高速率CAN数据采集, 以及CAN数据向车辆信号解析,和车辆信号黑白名单过滤, 以及MQTT上报云端,以及pprof性能观测,性能调优等功能。
做为一个Golang在车端嵌入式设备上应用的练手项目, 以供参考。