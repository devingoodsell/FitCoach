package pro.d11l.fitcoach.feature.settings

/** Formats a measurement for an editable text field: drops a trailing ".0" so a
 *  whole number prefills as "175" rather than "175.0". */
internal fun Double.asFieldText(): String =
    if (this % 1.0 == 0.0) toLong().toString() else toString()
