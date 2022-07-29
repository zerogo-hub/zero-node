package network

import "errors"

var (
	// ErrRouterRepeated 路由已存在
	ErrRouterRepeated = errors.New("router repeated")

	// ErrHandlerNotFound 处理函数未找到
	ErrHandlerNotFound = errors.New("handler not found")
)

type router struct {
	routes map[uint16]HandlerFunc

	// 自定义处理逻辑
	handlerFunc func(Message) (Message, error)
}

// NewRouter 创建一个路由器
func NewRouter() Router {
	return &router{
		routes: make(map[uint16]HandlerFunc),
	}
}

// RouterID 转化路由 Id
func RouterID(module, action uint8) uint16 {
	return uint16(module<<4 + action)
}

// AddRouter 添加路由
func (router *router) AddRouter(module, action uint8, handler HandlerFunc) error {
	if handler == nil {
		return errors.New("handle can not be nil")
	}

	routerID := RouterID(module, action)

	if _, ok := router.routes[routerID]; ok {
		return ErrRouterRepeated
	}

	router.routes[routerID] = handler

	return nil
}

// Handler 路由处理
func (router *router) Handler(message Message) (Message, error) {
	routerID := RouterID(message.ModuleID(), message.ActionID())

	handler, ok := router.routes[routerID]
	if !ok {
		if router.handlerFunc != nil {
			return router.handlerFunc(message)
		}

		return nil, ErrHandlerNotFound
	}

	return handler(message)
}

// SetHandlerFunc 设置自定义处理逻辑
func (router *router) SetHandlerFunc(handler HandlerFunc) {
	router.handlerFunc = handler
}
