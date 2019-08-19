// Copyright 2018 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"errors"

	pb "github.com/casbin/casbin-server/proto"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// Server is used to implement proto.CasbinServer.
type Server struct {
	enforcerMap map[int]*casbin.Enforcer
	adapter     persist.Adapter
}

// NewServer creates a new instance the server.
func NewServer(driverName string, connectionString string, dbSpecified bool) *Server {
	s := Server{}
	var err error

	s.adapter, err = newAdapter(driverName, connectionString, dbSpecified)
	if err != nil {
		panic(err)
	}
	s.enforcerMap = map[int]*casbin.Enforcer{}

	return &s
}

func (s *Server) getEnforcer(handle int) (*casbin.Enforcer, error) {
	if _, ok := s.enforcerMap[handle]; ok {
		return s.enforcerMap[handle], nil
	} else {
		return nil, errors.New("enforcer not found")
	}
}

func (s *Server) addEnforcer(e *casbin.Enforcer) int {
	cnt := len(s.enforcerMap)
	s.enforcerMap[cnt] = e
	return cnt
}

func (s *Server) NewEnforcer(ctx context.Context, in *pb.NewEnforcerRequest) (*pb.NewEnforcerReply, error) {
	var e *casbin.Enforcer

	if s.adapter == nil {
		m, err := model.NewModelFromString(in.ModelText)
		if err != nil {
			return &pb.NewEnforcerReply{Handler: 0}, err
		}

		e, err = casbin.NewEnforcer(m)
		if err != nil {
			return &pb.NewEnforcerReply{Handler: 0}, err
		}
	} else {
		m, err := model.NewModelFromString(in.ModelText)
		if err != nil {
			return &pb.NewEnforcerReply{Handler: 0}, err
		}

		e, err = casbin.NewEnforcer(m, s.adapter)
		if err != nil {
			return &pb.NewEnforcerReply{Handler: 0}, err
		}
	}
	h := s.addEnforcer(e)

	return &pb.NewEnforcerReply{Handler: int32(h)}, nil
}

func (s *Server) Enforce(ctx context.Context, in *pb.EnforceRequest) (*pb.BoolReply, error) {
	e, err := s.getEnforcer(int(in.EnforcerHandler))
	if err != nil {
		return &pb.BoolReply{Res: false}, err
	}

	params := make([]interface{}, 0, len(in.Params))

	m := e.GetModel()["m"]["m"]
	sourceValue := m.Value

	for index := range in.Params {
		param := parseAbacParam(in.Params[index], m)
		params = append(params, param)
	}

	res, err := e.Enforce(params...)
	if err != nil {
		return &pb.BoolReply{Res: false}, err
	}

	m.Value = sourceValue

	return &pb.BoolReply{Res: res}, nil
}

func (s *Server) LoadPolicy(ctx context.Context, in *pb.EmptyRequest) (*pb.EmptyReply, error) {
	e, err := s.getEnforcer(int(in.Handler))
	if err != nil {
		return &pb.EmptyReply{}, err
	}

	err = e.LoadPolicy()

	return &pb.EmptyReply{}, err
}

func (s *Server) SavePolicy(ctx context.Context, in *pb.EmptyRequest) (*pb.EmptyReply, error) {
	e, err := s.getEnforcer(int(in.Handler))
	if err != nil {
		return &pb.EmptyReply{}, err
	}

	err = e.SavePolicy()

	return &pb.EmptyReply{}, err
}
