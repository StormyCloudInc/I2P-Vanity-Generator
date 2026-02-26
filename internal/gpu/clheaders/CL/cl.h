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
 * Minimal vendored OpenCL 1.2 header.
 * Contains only the types, constants, and function declarations
 * required by this project.
 */

#ifndef CL_H
#define CL_H

#include "cl_platform.h"

#ifdef __cplusplus
extern "C" {
#endif

/* --- Opaque handle types ------------------------------------------------- */

typedef struct _cl_platform_id*    cl_platform_id;
typedef struct _cl_device_id*      cl_device_id;
typedef struct _cl_context*        cl_context;
typedef struct _cl_command_queue*   cl_command_queue;
typedef struct _cl_mem*            cl_mem;
typedef struct _cl_program*        cl_program;
typedef struct _cl_kernel*         cl_kernel;
typedef struct _cl_event*          cl_event;

/* --- Enum / flag types --------------------------------------------------- */

typedef cl_bitfield          cl_device_type;
typedef cl_uint              cl_device_info;
typedef cl_uint              cl_platform_info;
typedef cl_uint              cl_context_info;
typedef intptr_t             cl_context_properties;
typedef cl_bitfield          cl_command_queue_properties;
typedef cl_bitfield          cl_mem_flags;
typedef cl_uint              cl_program_build_info;
typedef cl_uint              cl_kernel_info;

/* --- Error codes --------------------------------------------------------- */

#define CL_SUCCESS                       0

/* --- cl_bool ------------------------------------------------------------- */

#define CL_TRUE                          1
#define CL_FALSE                         0

/* --- cl_device_type ------------------------------------------------------ */

#define CL_DEVICE_TYPE_GPU               (1 << 2)

/* --- cl_device_info ------------------------------------------------------ */

#define CL_DEVICE_NAME                   0x102B
#define CL_DEVICE_VENDOR                 0x102C

/* --- cl_mem_flags -------------------------------------------------------- */

#define CL_MEM_READ_WRITE                (1 << 0)
#define CL_MEM_READ_ONLY                 (1 << 2)
#define CL_MEM_COPY_HOST_PTR            (1 << 5)

/* --- cl_program_build_info ----------------------------------------------- */

#define CL_PROGRAM_BUILD_LOG             0x1183

/* --- Platform APIs ------------------------------------------------------- */

extern cl_int
clGetPlatformIDs(cl_uint          num_entries,
                 cl_platform_id*  platforms,
                 cl_uint*         num_platforms);

/* --- Device APIs --------------------------------------------------------- */

extern cl_int
clGetDeviceIDs(cl_platform_id   platform,
               cl_device_type   device_type,
               cl_uint          num_entries,
               cl_device_id*    devices,
               cl_uint*         num_devices);

extern cl_int
clGetDeviceInfo(cl_device_id    device,
                cl_device_info  param_name,
                size_t          param_value_size,
                void*           param_value,
                size_t*         param_value_size_ret);

/* --- Context APIs -------------------------------------------------------- */

typedef void (CL_CALLBACK *cl_context_callback)(
    const char* errinfo, const void* private_info,
    size_t cb, void* user_data);

extern cl_context
clCreateContext(const cl_context_properties* properties,
                cl_uint              num_devices,
                const cl_device_id*  devices,
                cl_context_callback  pfn_notify,
                void*                user_data,
                cl_int*              errcode_ret);

extern cl_int
clReleaseContext(cl_context context);

/* --- Command Queue APIs -------------------------------------------------- */

extern cl_command_queue
clCreateCommandQueue(cl_context                  context,
                     cl_device_id                device,
                     cl_command_queue_properties  properties,
                     cl_int*                     errcode_ret);

extern cl_int
clReleaseCommandQueue(cl_command_queue command_queue);

/* --- Memory Object APIs -------------------------------------------------- */

extern cl_mem
clCreateBuffer(cl_context   context,
               cl_mem_flags flags,
               size_t       size,
               void*        host_ptr,
               cl_int*      errcode_ret);

extern cl_int
clReleaseMemObject(cl_mem memobj);

/* --- Program Object APIs ------------------------------------------------- */

extern cl_program
clCreateProgramWithSource(cl_context        context,
                          cl_uint           count,
                          const char**      strings,
                          const size_t*     lengths,
                          cl_int*           errcode_ret);

typedef void (CL_CALLBACK *cl_program_callback)(
    cl_program program, void* user_data);

extern cl_int
clBuildProgram(cl_program           program,
               cl_uint              num_devices,
               const cl_device_id*  device_list,
               const char*          options,
               cl_program_callback  pfn_notify,
               void*                user_data);

extern cl_int
clGetProgramBuildInfo(cl_program            program,
                      cl_device_id          device,
                      cl_program_build_info param_name,
                      size_t                param_value_size,
                      void*                 param_value,
                      size_t*               param_value_size_ret);

extern cl_int
clReleaseProgram(cl_program program);

/* --- Kernel Object APIs -------------------------------------------------- */

extern cl_kernel
clCreateKernel(cl_program   program,
               const char*  kernel_name,
               cl_int*      errcode_ret);

extern cl_int
clSetKernelArg(cl_kernel    kernel,
               cl_uint      arg_index,
               size_t       arg_size,
               const void*  arg_value);

extern cl_int
clReleaseKernel(cl_kernel kernel);

/* --- Enqueued Commands APIs ---------------------------------------------- */

extern cl_int
clEnqueueReadBuffer(cl_command_queue command_queue,
                    cl_mem           buffer,
                    cl_bool          blocking_read,
                    size_t           offset,
                    size_t           size,
                    void*            ptr,
                    cl_uint          num_events_in_wait_list,
                    const cl_event*  event_wait_list,
                    cl_event*        event);

extern cl_int
clEnqueueWriteBuffer(cl_command_queue command_queue,
                     cl_mem           buffer,
                     cl_bool          blocking_write,
                     size_t           offset,
                     size_t           size,
                     const void*      ptr,
                     cl_uint          num_events_in_wait_list,
                     const cl_event*  event_wait_list,
                     cl_event*        event);

extern cl_int
clEnqueueNDRangeKernel(cl_command_queue command_queue,
                       cl_kernel        kernel,
                       cl_uint          work_dim,
                       const size_t*    global_work_offset,
                       const size_t*    global_work_size,
                       const size_t*    local_work_size,
                       cl_uint          num_events_in_wait_list,
                       const cl_event*  event_wait_list,
                       cl_event*        event);

/* --- Flush and Finish APIs ----------------------------------------------- */

extern cl_int
clFinish(cl_command_queue command_queue);

#ifdef __cplusplus
}
#endif

#endif /* CL_H */
