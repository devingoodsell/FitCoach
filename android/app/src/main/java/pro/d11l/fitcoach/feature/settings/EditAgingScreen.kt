package pro.d11l.fitcoach.feature.settings

import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle

@Composable
fun EditAgingScreen(viewModel: EditAgingViewModel, onDone: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }

    SettingsEditScaffold(
        title = "Aging emphases",
        isSaving = state.isSaving,
        onBack = onDone,
        onSave = viewModel::save,
        error = state.error,
    ) {
        Text(
            "These weight the healthspan focus your coach builds into each session. " +
                "They default from your age; adjust to your priorities.",
        )
        SettingsSliderRow("Bone & balance", state.boneBalance, viewModel::onBoneBalance)
        SettingsSliderRow("Joint & tendon", state.jointTendon, viewModel::onJointTendon)
        SettingsSliderRow("VO₂ max", state.vo2max, viewModel::onVo2max)
        SettingsSliderRow("Cardio base", state.cardioBase, viewModel::onCardioBase)
    }
}
