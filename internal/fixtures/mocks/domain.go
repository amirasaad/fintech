// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"github.com/amirasaad/fintech/pkg/domain/common"
	mock "github.com/stretchr/testify/mock"
)

// NewMockCurrencyConverter creates a new instance of MockCurrencyConverter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCurrencyConverter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCurrencyConverter {
	mock := &MockCurrencyConverter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockCurrencyConverter is an autogenerated mock type for the CurrencyConverter type
type MockCurrencyConverter struct {
	mock.Mock
}

type MockCurrencyConverter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockCurrencyConverter) EXPECT() *MockCurrencyConverter_Expecter {
	return &MockCurrencyConverter_Expecter{mock: &_m.Mock}
}

// Convert provides a mock function for the type MockCurrencyConverter
func (_mock *MockCurrencyConverter) Convert(amount float64, from string, to string) (*common.ConversionInfo, error) {
	ret := _mock.Called(amount, from, to)

	if len(ret) == 0 {
		panic("no return value specified for Convert")
	}

	var r0 *common.ConversionInfo
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(float64, string, string) (*common.ConversionInfo, error)); ok {
		return returnFunc(amount, from, to)
	}
	if returnFunc, ok := ret.Get(0).(func(float64, string, string) *common.ConversionInfo); ok {
		r0 = returnFunc(amount, from, to)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*common.ConversionInfo)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(float64, string, string) error); ok {
		r1 = returnFunc(amount, from, to)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockCurrencyConverter_Convert_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Convert'
type MockCurrencyConverter_Convert_Call struct {
	*mock.Call
}

// Convert is a helper method to define mock.On call
//   - amount float64
//   - from string
//   - to string
func (_e *MockCurrencyConverter_Expecter) Convert(amount interface{}, from interface{}, to interface{}) *MockCurrencyConverter_Convert_Call {
	return &MockCurrencyConverter_Convert_Call{Call: _e.mock.On("Convert", amount, from, to)}
}

func (_c *MockCurrencyConverter_Convert_Call) Run(run func(amount float64, from string, to string)) *MockCurrencyConverter_Convert_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 float64
		if args[0] != nil {
			arg0 = args[0].(float64)
		}
		var arg1 string
		if args[1] != nil {
			arg1 = args[1].(string)
		}
		var arg2 string
		if args[2] != nil {
			arg2 = args[2].(string)
		}
		run(
			arg0,
			arg1,
			arg2,
		)
	})
	return _c
}

func (_c *MockCurrencyConverter_Convert_Call) Return(conversionInfo *common.ConversionInfo, err error) *MockCurrencyConverter_Convert_Call {
	_c.Call.Return(conversionInfo, err)
	return _c
}

func (_c *MockCurrencyConverter_Convert_Call) RunAndReturn(run func(amount float64, from string, to string) (*common.ConversionInfo, error)) *MockCurrencyConverter_Convert_Call {
	_c.Call.Return(run)
	return _c
}

// GetRate provides a mock function for the type MockCurrencyConverter
func (_mock *MockCurrencyConverter) GetRate(from string, to string) (float64, error) {
	ret := _mock.Called(from, to)

	if len(ret) == 0 {
		panic("no return value specified for GetRate")
	}

	var r0 float64
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(string, string) (float64, error)); ok {
		return returnFunc(from, to)
	}
	if returnFunc, ok := ret.Get(0).(func(string, string) float64); ok {
		r0 = returnFunc(from, to)
	} else {
		r0 = ret.Get(0).(float64)
	}
	if returnFunc, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = returnFunc(from, to)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockCurrencyConverter_GetRate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRate'
type MockCurrencyConverter_GetRate_Call struct {
	*mock.Call
}

// GetRate is a helper method to define mock.On call
//   - from string
//   - to string
func (_e *MockCurrencyConverter_Expecter) GetRate(from interface{}, to interface{}) *MockCurrencyConverter_GetRate_Call {
	return &MockCurrencyConverter_GetRate_Call{Call: _e.mock.On("GetRate", from, to)}
}

func (_c *MockCurrencyConverter_GetRate_Call) Run(run func(from string, to string)) *MockCurrencyConverter_GetRate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		var arg1 string
		if args[1] != nil {
			arg1 = args[1].(string)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockCurrencyConverter_GetRate_Call) Return(f float64, err error) *MockCurrencyConverter_GetRate_Call {
	_c.Call.Return(f, err)
	return _c
}

func (_c *MockCurrencyConverter_GetRate_Call) RunAndReturn(run func(from string, to string) (float64, error)) *MockCurrencyConverter_GetRate_Call {
	_c.Call.Return(run)
	return _c
}

// IsSupported provides a mock function for the type MockCurrencyConverter
func (_mock *MockCurrencyConverter) IsSupported(from string, to string) bool {
	ret := _mock.Called(from, to)

	if len(ret) == 0 {
		panic("no return value specified for IsSupported")
	}

	var r0 bool
	if returnFunc, ok := ret.Get(0).(func(string, string) bool); ok {
		r0 = returnFunc(from, to)
	} else {
		r0 = ret.Get(0).(bool)
	}
	return r0
}

// MockCurrencyConverter_IsSupported_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsSupported'
type MockCurrencyConverter_IsSupported_Call struct {
	*mock.Call
}

// IsSupported is a helper method to define mock.On call
//   - from string
//   - to string
func (_e *MockCurrencyConverter_Expecter) IsSupported(from interface{}, to interface{}) *MockCurrencyConverter_IsSupported_Call {
	return &MockCurrencyConverter_IsSupported_Call{Call: _e.mock.On("IsSupported", from, to)}
}

func (_c *MockCurrencyConverter_IsSupported_Call) Run(run func(from string, to string)) *MockCurrencyConverter_IsSupported_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		var arg1 string
		if args[1] != nil {
			arg1 = args[1].(string)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockCurrencyConverter_IsSupported_Call) Return(b bool) *MockCurrencyConverter_IsSupported_Call {
	_c.Call.Return(b)
	return _c
}

func (_c *MockCurrencyConverter_IsSupported_Call) RunAndReturn(run func(from string, to string) bool) *MockCurrencyConverter_IsSupported_Call {
	_c.Call.Return(run)
	return _c
}
