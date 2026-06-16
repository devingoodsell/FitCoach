package pro.d11l.fitcoach.feature.injury

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.InjuryDto
import pro.d11l.fitcoach.data.InjuryRepository

/** Lifecycle statuses the UI offers. */
val INJURY_STATUSES = listOf("active_flare", "managed", "recurring_but_fine", "resolved")
val SEVERITIES = listOf("mild", "moderate", "severe")

data class InjuryUiState(
    val loading: Boolean = true,
    val injuries: List<InjuryDto> = emptyList(),
    // draft/edit form
    val freeText: String = "",
    val draftVisible: Boolean = false,
    val region: String = "",
    val status: String = "active_flare",
    val severity: String = "moderate",
    val aggravating: String = "",
    val notes: String = "",
    val lowConfidenceFields: List<String> = emptyList(),
    val saving: Boolean = false,
    val error: String? = null,
)

class InjuryViewModel(private val repo: InjuryRepository) : ViewModel() {

    private val _state = MutableStateFlow(InjuryUiState())
    val state: StateFlow<InjuryUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.load()
                .onSuccess { doc -> _state.update { it.copy(loading = false, injuries = doc.injuries) } }
                .onFailure { e -> _state.update { it.copy(loading = false, error = e.message) } }
        }
    }

    fun onFreeText(v: String) = _state.update { it.copy(freeText = v) }
    fun onRegion(v: String) = _state.update { it.copy(region = v) }
    fun onStatus(v: String) = _state.update { it.copy(status = v) }
    fun onSeverity(v: String) = _state.update { it.copy(severity = v) }
    fun onAggravating(v: String) = _state.update { it.copy(aggravating = v) }
    fun onNotes(v: String) = _state.update { it.copy(notes = v) }

    /** Start a blank manual draft. */
    fun startManual() = _state.update {
        it.copy(draftVisible = true, region = "", status = "active_flare", severity = "moderate",
            aggravating = "", notes = "", lowConfidenceFields = emptyList())
    }

    fun cancelDraft() = _state.update { it.copy(draftVisible = false, lowConfidenceFields = emptyList()) }

    /** Parse the freeform text into an editable draft for review before saving (E7-S1). */
    fun parse() {
        val text = _state.value.freeText.trim()
        if (text.isEmpty()) {
            _state.update { it.copy(error = "Describe the injury first") }
            return
        }
        viewModelScope.launch {
            repo.parse(text)
                .onSuccess { d ->
                    _state.update {
                        it.copy(
                            draftVisible = true,
                            region = d.injury.region,
                            status = d.injury.status.ifEmpty { "active_flare" },
                            severity = d.injury.severity.ifEmpty { "moderate" },
                            aggravating = d.injury.aggravatingMovements.joinToString(", "),
                            notes = d.injury.notes,
                            lowConfidenceFields = d.lowConfidenceFields,
                        )
                    }
                }
                .onFailure { e -> _state.update { it.copy(error = e.message) } }
        }
    }

    /** Save the reviewed draft as a new injury. */
    fun saveDraft() {
        val s = _state.value
        if (s.region.isBlank()) {
            _state.update { it.copy(error = "Region is required") }
            return
        }
        _state.update { it.copy(saving = true, error = null) }
        viewModelScope.launch {
            val dto = InjuryDto(
                region = s.region.trim(),
                status = s.status,
                severity = s.severity,
                aggravatingMovements = parseList(s.aggravating),
                notes = s.notes.trim(),
            )
            repo.add(dto)
                .onSuccess {
                    _state.update {
                        it.copy(saving = false, draftVisible = false, freeText = "", region = "",
                            aggravating = "", notes = "", lowConfidenceFields = emptyList())
                    }
                    load()
                }
                .onFailure { e -> _state.update { it.copy(saving = false, error = e.message) } }
        }
    }

    fun setStatus(injury: InjuryDto, status: String) {
        viewModelScope.launch {
            repo.update(injury.id, injury.copy(status = status))
                .onSuccess { load() }
                .onFailure { e -> _state.update { it.copy(error = e.message) } }
        }
    }

    fun delete(id: String) {
        viewModelScope.launch {
            repo.delete(id)
                .onSuccess { load() }
                .onFailure { e -> _state.update { it.copy(error = e.message) } }
        }
    }

    private fun parseList(raw: String): List<String> =
        raw.split(",").map(String::trim).filter(String::isNotEmpty)
}
