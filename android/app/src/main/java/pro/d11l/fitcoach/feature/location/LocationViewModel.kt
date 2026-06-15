package pro.d11l.fitcoach.feature.location

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.CurrentContextDto
import pro.d11l.fitcoach.core.network.LocationDto
import pro.d11l.fitcoach.data.LocationRepository
import pro.d11l.fitcoach.data.LocationResult

data class LocationUiState(
    val loading: Boolean = true,
    val locations: List<LocationDto> = emptyList(),
    val current: CurrentContextDto? = null,
    val error: String? = null,
)

class LocationViewModel(private val repo: LocationRepository) : ViewModel() {

    private val _state = MutableStateFlow(LocationUiState())
    val state: StateFlow<LocationUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            when (val r = repo.load()) {
                is LocationResult.Ok ->
                    _state.update { it.copy(loading = false, locations = r.value.locations, current = r.value.currentContext) }
                is LocationResult.Error ->
                    _state.update { it.copy(loading = false, error = r.message) }
            }
        }
    }

    /** equipment is a comma-separated string from the UI. */
    fun addLocation(name: String, equipment: String) {
        if (name.isBlank()) {
            _state.update { it.copy(error = "Name is required") }
            return
        }
        viewModelScope.launch {
            when (val r = repo.add(name.trim(), parseEquipment(equipment))) {
                is LocationResult.Ok -> load()
                is LocationResult.Error -> _state.update { it.copy(error = r.message) }
            }
        }
    }

    fun deleteLocation(id: String) {
        viewModelScope.launch {
            when (val r = repo.delete(id)) {
                is LocationResult.Ok -> load()
                is LocationResult.Error -> _state.update { it.copy(error = r.message) }
            }
        }
    }

    fun setCurrent(locationId: String, note: String) {
        viewModelScope.launch {
            when (val r = repo.setCurrent(locationId, note)) {
                is LocationResult.Ok -> _state.update { it.copy(current = r.value) }
                is LocationResult.Error -> _state.update { it.copy(error = r.message) }
            }
        }
    }

    private fun parseEquipment(raw: String): List<String> =
        raw.split(",").map(String::trim).filter(String::isNotEmpty)
}
