/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rbac

import (
	"fmt"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	repository2 "github.com/devtron-labs/devtron/pkg/appStore/installedApp/repository"
	"github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/team"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type EnforcerUtilHelm interface {
	GetHelmObjectByClusterId(clusterId int, namespace string, appName string) string
	GetHelmObjectByTeamIdAndClusterId(teamId int, clusterId int, namespace string, appName string) string
	GetHelmObjectByClusterIdNamespaceAndAppName(clusterId int, namespace string, appName string) (string, string)
	GetAppRBACNameByInstalledAppId(installedAppId int) (string, string)
	GetAppRBACNameByInstalledAppIdAndTeamId(installedAppId, teamId int) string
}
type EnforcerUtilHelmImpl struct {
	logger                 *zap.SugaredLogger
	clusterRepository      repository.ClusterRepository
	teamRepository         team.TeamRepository
	appRepository          app.AppRepository
	InstalledAppRepository repository2.InstalledAppRepository
}

func NewEnforcerUtilHelmImpl(logger *zap.SugaredLogger,
	clusterRepository repository.ClusterRepository,
	teamRepository team.TeamRepository,
	appRepository app.AppRepository,
	installedAppRepository repository2.InstalledAppRepository,
) *EnforcerUtilHelmImpl {
	return &EnforcerUtilHelmImpl{
		logger:                 logger,
		clusterRepository:      clusterRepository,
		teamRepository:         teamRepository,
		appRepository:          appRepository,
		InstalledAppRepository: installedAppRepository,
	}
}

func (impl EnforcerUtilHelmImpl) GetHelmObjectByClusterId(clusterId int, namespace string, appName string) string {
	cluster, err := impl.clusterRepository.FindById(clusterId)
	if err != nil {
		return fmt.Sprintf("%s/%s/%s", "", "", "")
	}
	return fmt.Sprintf("%s/%s__%s/%s", team.UNASSIGNED_PROJECT, cluster.ClusterName, namespace, appName)
}

func (impl EnforcerUtilHelmImpl) GetHelmObjectByTeamIdAndClusterId(teamId int, clusterId int, namespace string, appName string) string {

	cluster, err := impl.clusterRepository.FindById(clusterId)

	teamObj, err := impl.teamRepository.FindOne(teamId)

	if err != nil {
		return fmt.Sprintf("%s/%s/%s", "", "", "")
	}
	return fmt.Sprintf("%s/%s__%s/%s", teamObj.Name, cluster.ClusterName, namespace, appName)
}

func (impl EnforcerUtilHelmImpl) GetHelmObjectByClusterIdNamespaceAndAppName(clusterId int, namespace string, appName string) (string, string) {

	installedApp, installedAppErr := impl.InstalledAppRepository.GetInstalledApplicationByClusterIdAndNamespaceAndAppName(clusterId, namespace, appName)

	if installedAppErr != nil && installedAppErr != pg.ErrNoRows {
		impl.logger.Errorw("error on fetching data for rbac object from installed app repository", "err", installedAppErr)
		return "", ""
	}

	cluster, err := impl.clusterRepository.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error on fetching data for rbac object from cluster repository", "err", err)
		return "", ""
	}

	if installedApp == nil || installedAppErr == pg.ErrNoRows {
		// for cli apps which are not yet linked

		app, err := impl.appRepository.FindAppAndProjectByAppName(appName)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching app details", "err", err)
			return "", ""
		}

		if app.TeamId == 0 {
			// case if project is not assigned to cli app
			return fmt.Sprintf("%s/%s__%s/%s", team.UNASSIGNED_PROJECT, cluster.ClusterName, namespace, appName), ""
		} else {
			// case if project is assigned
			return fmt.Sprintf("%s/%s__%s/%s", app.Team.Name, cluster.ClusterName, namespace, appName), ""
		}

	}

	if installedApp.App.TeamId == 0 {
		// for EA apps which have no project assigned to them
		return fmt.Sprintf("%s/%s__%s/%s", team.UNASSIGNED_PROJECT, cluster.ClusterName, namespace, appName),
			fmt.Sprintf("%s/%s/%s", team.UNASSIGNED_PROJECT, installedApp.Environment.EnvironmentIdentifier, appName)

	} else {
		if installedApp.EnvironmentId == 0 {
			// for apps in EA mode, initally env can be 0.
			return fmt.Sprintf("%s/%s__%s/%s", installedApp.App.Team.Name, cluster.ClusterName, namespace, appName), ""
		}
		// for apps which are assigned to a project and have env ID
		rbacOne := fmt.Sprintf("%s/%s/%s", installedApp.App.Team.Name, installedApp.Environment.EnvironmentIdentifier, appName)
		rbacTwo := fmt.Sprintf("%s/%s__%s/%s", installedApp.App.Team.Name, cluster.ClusterName, namespace, appName)
		if installedApp.Environment.IsVirtualEnvironment {
			return rbacOne, ""
		}
		return rbacOne, rbacTwo
	}

}

func (impl EnforcerUtilHelmImpl) GetAppRBACNameByInstalledAppId(installedAppVersionId int) (string, string) {

	InstalledApp, err := impl.InstalledAppRepository.GetInstalledApp(installedAppVersionId)
	if err != nil {
		impl.logger.Errorw("error in fetching installed app version data", "err", err)
		return fmt.Sprintf("%s/%s/%s", "", "", ""), fmt.Sprintf("%s/%s/%s", "", "", "")
	}
	rbacOne := fmt.Sprintf("%s/%s/%s", InstalledApp.App.Team.Name, InstalledApp.Environment.EnvironmentIdentifier, InstalledApp.App.AppName)

	if InstalledApp.Environment.IsVirtualEnvironment {
		return rbacOne, ""
	}

	var rbacTwo string
	if !InstalledApp.Environment.IsVirtualEnvironment {
		if InstalledApp.Environment.EnvironmentIdentifier != InstalledApp.Environment.Cluster.ClusterName+"__"+InstalledApp.Environment.Namespace {
			rbacTwo = fmt.Sprintf("%s/%s/%s", InstalledApp.App.Team.Name, InstalledApp.Environment.Cluster.ClusterName+"__"+InstalledApp.Environment.Namespace, InstalledApp.App.AppName)
			return rbacOne, rbacTwo
		}
	}

	return rbacOne, ""
}

func (impl EnforcerUtilHelmImpl) GetAppRBACNameByInstalledAppIdAndTeamId(installedAppId, teamId int) string {
	installedApp, err := impl.InstalledAppRepository.GetInstalledApp(installedAppId)
	if err != nil {
		impl.logger.Errorw("error in fetching installed app version data", "err", err)
		return fmt.Sprintf("%s/%s/%s", "", "", "")
	}
	project, err := impl.teamRepository.FindOne(teamId)
	if err != nil {
		impl.logger.Errorw("error in fetching project by teamID", "err", err)
		return fmt.Sprintf("%s/%s/%s", "", "", "")
	}
	rbac := fmt.Sprintf("%s/%s/%s", project.Name, installedApp.Environment.EnvironmentIdentifier, installedApp.App.AppName)
	return rbac
}
