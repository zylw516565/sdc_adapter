package dbc

type SgVO struct {
	/**
	* 信号的名字
	 */
	SignalName string
	/**
	* 多路选择信号的定义
	* a）空，表示普通信号。
	* b）M，表示多路选择器信号。
	* c）m50，表示被多路选择器选择的信号，50，表示当M定义的信号的值等于50的时候，该报文使用此通路。
	 */
	SigTypeDefinition string
	/**
	* 信号起始位
	 */
	StartBit int
	/**
	* 信号长度
	 */
	BitWidth int
	/**
	* 信号的字节顺序：0代表Motorola格式，1代表Inter格式
	 */
	ByteOrder int
	/**
	* 信号的数值类型：+表示无符号数，-表示有符号数；
	 */
	ValueType string
	/**
	* 因子
	 */
	Factor float64
	/**
	* 偏移量
	* 物理值=原始值*因子+偏移量；
	 */
	Offsets int64
	/**
	* 最小值
	 */
	Min float64
	/**
	* 最大值
	 */
	Max float64
	/**
	* 单位
	 */
	Unit string
	/**
	* 信号的接收节点 没有指定的接收节点，则必须设置为” Vector__XXX”
	 */
	Receiver string
	/**
	* 注解里面的描述
	 */
	Desc string
	/**
	* 注解内容
	 */
	Comment string
	/**
	* val参数
	 */
	DefineMap map[string]string
	/**
	* dbc 原文
	 */
	DbcContent string
}

type BoVO struct {
	/**
	* 报文id
	 */
	CanId uint64
	/**
	* 报文的名字
	 */
	CanName string
	/**
	* 报文长度(字节)
	 */
	DataLenth uint64
	/**
	* 报文发送节点
	 */
	Transmitter string
	/**
	* 下级信号，信号name为key
	 */
	SgVoMap map[string]*SgVO
	/**
	* 信号名列表,按DBC文件内信号顺序存放
	 */
	OrderedSignals []string
	/**
	* 注解里面的描述
	 */
	Desc string
	/**
	* 注解内容
	 */
	Comment string
	/**
	* dbc 原文
	 */
	DbcContent string
}

type CmVO struct {
	/**
	* Object：表示进行注解的对象类型，可以是节点“BU_”、报文“BO_”、消息”SG_”
	 */
	ObjectType string
	/**
	* 报文id  消息或者报文类型
	 */
	CanId uint64
	/**
	*  节点名称   can名称  信号名称
	 */
	Name string
	/**
	*  描述
	 */
	Desc string
	/**
	* 注解的信息
	 */
	Comment string
	/**
	* dbc 原文
	 */
	DbcContent string
}

type ValVO struct {
	/**
	 * 报文id  消息或者报文类型
	 */
	CanId uint64
	/**
	 * 信号的名字
	 */
	SignalName string
	/**
	 * 数值 定义
	 */
	DefineMap map[string]string
	/**
	 * dbc 原文
	 */
	DbcContent string
}

type ValTableVO struct {
	/**
	 * 全局信号名称
	 */
	SignalName string
	/**
	 * 定义值
	 */
	DefinedMap map[string]string
}

type DbcVO struct {
	/**
	* 版本
	 */
	Version string
	/**
	* 新符号
	 */
	Ns []string
	/**
	* 波特率
	 */
	Bs string
	/**
	* 网络节点
	 */
	Bu []string
	/**
	* 报文帧定义 BO_标识, <canid, BO>
	 */
	BoVoMap map[uint64]*BoVO
	/**
	* 关键字，表示注解信息
	 */
	CmVOList []CmVO
	/**
	* 信号数值表的定义
	 */
	ValVOList []ValVO
	/**
	* 全局信号值
	 */
	ValTableVOList []ValTableVO
	/**
	* 未解析字段
	 */
	UnParseLines []string
}
