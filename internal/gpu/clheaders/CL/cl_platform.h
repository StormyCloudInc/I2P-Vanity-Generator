/*
 * Copyright (c) 2008-2024 The Khronos Group Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Minimal vendored OpenCL 1.2 platform header.
 * Contains only the type definitions required by this project.
 */

#ifndef CL_PLATFORM_H
#define CL_PLATFORM_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Scalar types */
typedef int8_t   cl_char;
typedef uint8_t  cl_uchar;
typedef int16_t  cl_short;
typedef uint16_t cl_ushort;
typedef int32_t  cl_int;
typedef uint32_t cl_uint;
typedef int64_t  cl_long;
typedef uint64_t cl_ulong;
typedef uint16_t cl_half;
typedef float    cl_float;
typedef double   cl_double;

/* Boolean type */
typedef cl_uint  cl_bool;

/* Bit field type */
typedef cl_ulong cl_bitfield;

/* Calling convention qualifier for callback functions */
#define CL_CALLBACK

#ifdef __cplusplus
}
#endif

#endif /* CL_PLATFORM_H */
