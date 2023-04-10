package component

// Component 组件，如网关组件
type Component interface {
	// ID 每一个组件都有一个唯一编号
	ID() uint64
	// Name 获取组件名称
	Name() string
	// Version 版本号，用于更新时使用
	Version() string
	// Init 进行初始化，当尚未启动对外组件
	Init() error
	// Start 启动组件，可以接入新数据
	Start() error
	// Close 关闭组件
	Close() error
	// Resume 对于已经 pause 的组件进行恢复
	Resume() error
	// Pause 暂停组件，不可以接入新数据，但会继续处理已存在的数据
	Pause() error
}
